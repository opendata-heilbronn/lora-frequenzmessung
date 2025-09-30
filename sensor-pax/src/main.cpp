#define LoRaWAN_DEBUG_LEVEL 0
#define uS_TO_S_FACTOR 1000000ULL

#include <Arduino.h>
#include "HT_lCMEN2R13EFC1.h"

#include "logging.h"
#include "lora.h"
#include "pax.h"
#include "display.h"

// --- Battery Configuration ---
// Heltec V3 hardware: voltage divider (390k + 100k), battery ADC on GPIO1, control on GPIO21
#define VBAT_ADC_CTL 21
const int VBAT_ADC_PIN = 1;
const float VOLTAGE_DIVIDER_RATIO = 4.9;  // (390k + 100k) / 100k

// LiPo battery voltage thresholds
const float BATTERY_MAX_VOLTAGE = 4.2;  // 100%
const float BATTERY_MIN_VOLTAGE = 3.3;  // 0%

float readBatteryVoltage() {
  // Enable the battery voltage divider only during measurement
  pinMode(VBAT_ADC_CTL, OUTPUT);
  digitalWrite(VBAT_ADC_CTL, HIGH); // enable
  delay(50); // settle time for high impedance divider

  // Dummy read then real read for accurate value
  (void)analogReadMilliVolts(VBAT_ADC_PIN);
  delay(5);
  int analogVolts = analogReadMilliVolts(VBAT_ADC_PIN);

  // Disable the divider immediately after sampling; keep OUTPUT LOW when idle
  digitalWrite(VBAT_ADC_CTL, LOW);

  // Convert measured pin voltage (mV) to actual battery voltage (V)
  float batteryVoltage = (analogVolts / 1000.0f) * VOLTAGE_DIVIDER_RATIO;
  return batteryVoltage;
}

float calculateBatteryPercentage(float voltage) {
  if (voltage > BATTERY_MAX_VOLTAGE) voltage = BATTERY_MAX_VOLTAGE;
  if (voltage < BATTERY_MIN_VOLTAGE) voltage = BATTERY_MIN_VOLTAGE;

  float percentage = ((voltage - BATTERY_MIN_VOLTAGE) / (BATTERY_MAX_VOLTAGE - BATTERY_MIN_VOLTAGE)) * 100.0;

  return percentage;
}
void setup()
{
  Serial.begin(115200);
  delay(1000); // Take some time to open up the Serial Monitor

  esp_sleep_enable_timer_wakeup(sleepTime * uS_TO_S_FACTOR);
  logMessage("Setup ESP32 to sleep for every " + String(sleepTime) + " Seconds");

  InitPAX();
  displayMcuInit();
  InitLORA();

  // Battery ADC setup: 12-bit resolution, control pin idle LOW
  analogReadResolution(12);
  pinMode(VBAT_ADC_CTL, OUTPUT);
  digitalWrite(VBAT_ADC_CTL, LOW);
  
  firstrun = true;
}

void loop()
{
  float batteryPercentageLinear = 0;

  // Only read battery right before sending to avoid affecting JOIN/INIT
  if (deviceState == DEVICE_STATE_SEND) {
    float batteryVoltage = readBatteryVoltage();
    batteryPercentageLinear = calculateBatteryPercentage(batteryVoltage);
  }

  if (current_count != 0)
  {
    LoopLORA(current_count, batteryPercentageLinear);
  }
  
  firstrun = false;
}
