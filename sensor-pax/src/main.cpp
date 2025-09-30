#define LoRaWAN_DEBUG_LEVEL 0

#define uS_TO_S_FACTOR 1000000ULL
// --- Battery Voltage Configuration ---
// The voltage divider on the Heltec V3 board uses a 390k and 100k resistor, 
// so the measured voltage must be multiplied by 4.9 (390+100)/100.
const double VOLTAGE_DIVIDER_FACTOR = 4.9; 

// Define the voltage range for a standard LiPo battery.
const double MAX_BATT_VOLTAGE = 4.2; // Voltage for 100%
const double MIN_BATT_VOLTAGE = 3.0; // Voltage for 0%
#define VBAT_ADC_CTL 21

#include <Arduino.h>
#include "HT_lCMEN2R13EFC1.h"

#include "lora.h"
#include "pax.h"
#include "display.h"
// Battery voltage thresholds for LiPo battery
const float BATTERY_MAX_VOLTAGE = 4.2;
const float BATTERY_MIN_VOLTAGE = 3.3;
const float BATTERY_NOMINAL = 3.7;

// Voltage divider configuration for Heltec V3
const float R1 = 390.0;  // 390k ohm
const float R2 = 100.0;  // 100k ohm
const float VOLTAGE_DIVIDER_RATIO = (R1 + R2) / R2;  // 4.9

const float ADC_REFERENCE_VOLTAGE = 3.3;
const int ADC_RESOLUTION = 4096;  // 12-bit ADC
const int VBAT_ADC_PIN = 1;  // Heltec V3: battery divider output to ADC on GPIO1

double getBatteryValue() {
  // Enable the battery voltage divider only during measurement
  pinMode(VBAT_ADC_CTL, OUTPUT);
  digitalWrite(VBAT_ADC_CTL, HIGH); // enable divider (per Heltec example)
  delay(50); // allow RC network to settle (high impedance source)

  // Dummy read to charge ADC sampling capacitor
  (void)analogReadMilliVolts(VBAT_ADC_PIN);
  delay(5);

  int analogVolts = analogReadMilliVolts(VBAT_ADC_PIN);

  // Disable the divider to avoid interference/power drain
  digitalWrite(VBAT_ADC_CTL, LOW); // disable divider; keep OUTPUT LOW when idle

  // Convert measured pin voltage to actual battery voltage using divider factor
  return analogVolts * 490 / 100; // in millivolts
}

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
  // Clamp voltage to valid range
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
  Serial.println("Setup ESP32 to sleep for every " + String(sleepTime) + " Seconds");

  InitPAX();
  displayMcuInit();
  InitLORA();

  // Battery setup (not required if using getBatteryValue/readBatteryVoltage which handle it)
  analogReadResolution(12);
  // Ensure VBAT control pin idles LOW (per Heltec docs)
  pinMode(VBAT_ADC_CTL, OUTPUT);
  digitalWrite(VBAT_ADC_CTL, LOW);
  firstrun = true;
  // Defer battery measurement until just before sending to avoid any impact on join
  //double battery_voltage_mv = getBatteryValue();
  //Serial.printf("Battery Voltage: %.2fV", battery_voltage_mv / 1000.0);

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
    LoopLORA(current_count,batteryPercentageLinear);
  }
  else{
   // Serial.println("0 ergebniss ");
  }
  if (!firstrun)
  {
  //  esp_deep_sleep_start();
  }
    firstrun = false;

  //sleep(10);
}
