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
double getBatteryValue() {
  // Set the resolution of the analog-to-digital converter (ADC) to 12 bits (0-4095):
  analogReadResolution(12);

  // Set pin 37 as an output pin (used for ADC control):
  pinMode(37, OUTPUT);

  // Set pin 37 to HIGH (enable ADC control):
  digitalWrite(37, HIGH);
  int analogValue = analogRead(1);

  int analogVolts = analogReadMilliVolts(1);

  return analogVolts * 490 / 100;
}
float readBatteryVoltage() {
  int total = 0;
  const int numReadings = 10;

  for(int i = 0; i < numReadings; i++) {
    total += analogRead(1);
    delay(10);
  }
  int averageReading = total / numReadings;
  float adcVoltage = (averageReading * ADC_REFERENCE_VOLTAGE) / ADC_RESOLUTION;
  float batteryVoltage = adcVoltage * VOLTAGE_DIVIDER_RATIO;

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

  //batery settup
  //analogReadResolution(12);
  //pinMode(37, OUTPUT);
  //digitalWrite(37, HIGH);
  firstrun = true;
  //double battery_voltage = getBatteryValue();
  //Serial.printf("Battery Voltage: %.2fV", battery_voltage);

}

void loop()
{
  //float batteryVoltage = readBatteryVoltage();
  //float batteryPercentageLinear = calculateBatteryPercentage(batteryVoltage);
  //Serial.printf("Battery Percentage (Linear): %.1f%%\n", batteryPercentageLinear);
  float batteryPercentageLinear= 0;
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
