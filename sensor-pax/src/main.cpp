/**
 * Sensor-PAX Energy Efficient Implementation
 *
 * Architecture: Scan-on-Demand with Deep Sleep
 * [WAKE] -> [BLE SCAN 30s] -> [LoRa TX] -> [DEEP SLEEP ~870s]
 *
 * Power budget:
 * - Deep sleep: ~7uA
 * - BLE scanning: ~50mA @ 50% duty cycle
 * - Estimated daily consumption: ~47 mAh (vs ~2400 mAh continuous)
 */

#define LoRaWAN_DEBUG_LEVEL 0
#define uS_TO_S_FACTOR 1000000ULL

#include <Arduino.h>
#include "HT_lCMEN2R13EFC1.h"

#include "logging.h"
#include "lora.h"
#include "pax.h"
#include "display.h"
#include "mesh_comms.h"

// Define global variables from customs.h
char sensor_id[] = "863f75b0";
uint8_t devEui[] = {0x70, 0xB3, 0xD5, 0x7E, 0xD0, 0x07, 0x2D, 0x3C};
uint8_t appEui[] = {0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01};
uint8_t appKey[] = {0xC3, 0xD6, 0x99, 0x69, 0x1F, 0x7B, 0x58, 0x9A, 0x83, 0xEC, 0x4E, 0x28, 0x85, 0xAF, 0xE9, 0x62};
uint8_t nwkSKey[] = {0x08, 0xF6, 0xA5, 0x8E, 0x20, 0xE3, 0xAC, 0x24, 0x95, 0x1F, 0xFD, 0xCE, 0x03, 0x7E, 0x9A, 0x48};
uint8_t appSKey[] = {0x12, 0x08, 0x61, 0xE5, 0x38, 0x60, 0x56, 0xE6, 0xC1, 0xE9, 0x6B, 0x2C, 0x97, 0xC4, 0x2E, 0xEE};
uint32_t devAddr = (uint32_t)0x260BE3E9;
uint16_t userChannelsMask[6] = {0x00FF, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000};
float factor = 0.7;
float sleepTime = 900;
int SensorTypFrequency = 0;
int SensorTypBattery = 1;

// --- Battery Configuration ---
// Heltec V3 hardware: voltage divider (390k + 100k), battery ADC on GPIO1, control on GPIO37
#define VBAT_ADC_CTL 37
const int VBAT_ADC_PIN = 1;
const float VOLTAGE_DIVIDER_RATIO = 4.9;  // (390k + 100k) / 100k

// LiPo battery voltage thresholds
const float BATTERY_MAX_VOLTAGE = 4.2;  // 100%
const float BATTERY_MIN_VOLTAGE = 3.3;  // 0%

// --- Deep Sleep Configuration ---
// Calculate actual sleep time: total cycle - scan duration
const uint32_t ACTUAL_SLEEP_SEC = (uint32_t)sleepTime - BLE_SCAN_DURATION_SEC;

float readBatteryVoltage() {
    pinMode(VBAT_ADC_CTL, OUTPUT);
    digitalWrite(VBAT_ADC_CTL, HIGH);  // Enable voltage divider
    delay(10);

    // Dummy read then real read for accurate value
    (void)analogReadMilliVolts(VBAT_ADC_PIN);
    delay(2);
    int analogVolts = analogReadMilliVolts(VBAT_ADC_PIN);

    digitalWrite(VBAT_ADC_CTL, LOW);  // Disable to save power

    float batteryVoltage = (analogVolts / 1000.0f) * VOLTAGE_DIVIDER_RATIO;
    return batteryVoltage;
}

float calculateBatteryPercentage(float voltage) {
    if (voltage > BATTERY_MAX_VOLTAGE) voltage = BATTERY_MAX_VOLTAGE;
    if (voltage < BATTERY_MIN_VOLTAGE) voltage = BATTERY_MIN_VOLTAGE;

    float percentage = ((voltage - BATTERY_MIN_VOLTAGE) /
                        (BATTERY_MAX_VOLTAGE - BATTERY_MIN_VOLTAGE)) * 100.0;
    return percentage;
}

void enterDeepSleep() {
    logMessage("Entering deep sleep for " + String(ACTUAL_SLEEP_SEC) + " seconds...");

    // Ensure serial output is flushed
    Serial.flush();
    delay(10);

    // Configure wake-up timer
    esp_sleep_enable_timer_wakeup(ACTUAL_SLEEP_SEC * uS_TO_S_FACTOR);

    // Enter deep sleep
    esp_deep_sleep_start();

    // Code never reaches here
}

void setup() {
    Serial.begin(115200);
    delay(100);

    logMessage("=== Sensor-PAX Wake Cycle ===");
    logMessage("Cycle time: " + String((int)sleepTime) + "s (scan: " +
               String(BLE_SCAN_DURATION_SEC) + "s, sleep: " +
               String(ACTUAL_SLEEP_SEC) + "s)");

    // Initialize ADC for battery reading
    analogReadResolution(12);
    pinMode(VBAT_ADC_CTL, OUTPUT);
    digitalWrite(VBAT_ADC_CTL, LOW);  // Keep disabled until needed

#if ENABLE_DISPLAY
    displayMcuInit();
#endif

    // Initialize LoRa (must be done before BLE to avoid conflicts)
    InitLORA();

    // Initialize BLE scanner
    InitPAX();

    firstrun = true;
}

void loop() {
    // === Phase 1: BLE Scanning ===
    int pax_count = ScanPAX();
    logMessage("PAX count: " + String(pax_count));

    // === Phase 2: Deinitialize BLE before LoRa TX ===
    // This releases BLE resources and avoids conflicts
    DeinitPAX();

    // === Phase 3: Read Battery ===
    float batteryVoltage = readBatteryVoltage();
    float batteryPercentage = calculateBatteryPercentage(batteryVoltage);
    logMessage("Battery: " + String(batteryVoltage, 2) + "V (" +
               String(batteryPercentage, 1) + "%)");

    // === Phase 4: Dual-Mode Communication (Mesh-First with LoRaWAN Fallback) ===
    if (pax_count >= MIN_PAX_TO_SEND) {
        // Apply factor to get estimated actual people
        int estimated_pax = (int)(pax_count * factor);
        logMessage("Estimated PAX: " + String(estimated_pax));

        // --- Try Mesh Communication First ---
        logMessage("--- Attempting mesh transmission ---");
        bool meshSuccess = tryMeshTransmit(pax_count, batteryPercentage);

        if (meshSuccess) {
            logMessage("*** Mesh transmission successful ***");
            logMessage("Data sent via mesh network");

            // Cleanup mesh radio
            deinitMeshComms();
        } else {
            logMessage("*** Mesh transmission failed ***");
            logMessage("--- Falling back to LoRaWAN ---");

            // Cleanup mesh radio before switching to LoRaWAN
            deinitMeshComms();

            // Small delay to ensure radio state is clean
            delay(100);

            // Run LoRaWAN state machine until transmission completes
            // The state machine handles INIT -> JOIN -> SEND -> CYCLE -> SLEEP
            while (deviceState != DEVICE_STATE_SLEEP) {
                LoopLORA(pax_count, batteryPercentage);

                // Small delay to prevent tight loop
                delay(10);
            }

            logMessage("*** LoRaWAN transmission complete (fallback) ***");
        }
    } else {
        logMessage("PAX count < " + String(MIN_PAX_TO_SEND) +
                   ", skipping transmission for privacy");
    }

    firstrun = false;

    // === Phase 5: Deep Sleep ===
    enterDeepSleep();

    // Code never reaches here after deep sleep
}
