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

    // === Phase 4: LoRa Transmission ===
    if (pax_count >= MIN_PAX_TO_SEND) {
        // Apply factor to get estimated actual people
        int estimated_pax = (int)(pax_count * factor);
        logMessage("Transmitting estimated PAX: " + String(estimated_pax));

        // Run LoRaWAN state machine until transmission completes
        // The state machine handles INIT -> JOIN -> SEND -> CYCLE -> SLEEP
        while (deviceState != DEVICE_STATE_SLEEP) {
            LoopLORA(pax_count, batteryPercentage);

            // Small delay to prevent tight loop
            delay(10);
        }

        logMessage("LoRa transmission complete");
    } else {
        logMessage("PAX count < " + String(MIN_PAX_TO_SEND) +
                   ", skipping transmission for privacy");
    }

    firstrun = false;

    // === Phase 5: Deep Sleep ===
    enterDeepSleep();

    // Code never reaches here after deep sleep
}
