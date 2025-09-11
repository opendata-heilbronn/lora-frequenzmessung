#define LoRaWAN_DEBUG_LEVEL 0

#define uS_TO_S_FACTOR 1000000ULL
// --- Battery Voltage Configuration ---
// The voltage divider on the Heltec V3 board uses a 390k and 100k resistor, 
// so the measured voltage must be multiplied by 4.9 (390+100)/100.
const double VOLTAGE_DIVIDER_FACTOR = 4.9; 

// Define the voltage range for a standard LiPo battery.
const double MAX_BATT_VOLTAGE = 4.2; // Voltage for 100%
const double MIN_BATT_VOLTAGE = 3.0; // Voltage for 0%
#include <Arduino.h>
#include "HT_lCMEN2R13EFC1.h"

#include "lora.h"
#include "pax.h"
#include "display.h"

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
  analogReadResolution(12);
  pinMode(37, OUTPUT);
  digitalWrite(37, HIGH);
  firstrun = true;
}

void loop()
{ 
  int adc_millivolts = analogReadMilliVolts(1);
  double battery_voltage = (adc_millivolts / 1000.0) * VOLTAGE_DIVIDER_FACTOR;
  double battery_percentage = ((battery_voltage - MIN_BATT_VOLTAGE) / (MAX_BATT_VOLTAGE - MIN_BATT_VOLTAGE)) * 100.0;
  battery_percentage = constrain(battery_percentage, 0.0, 100.0);
  //Serial.printf("Battery Voltage: %.2fV,  Percentage: %.2f%%\n", battery_voltage, battery_percentage);
  if (current_count != 0)
  {    
    LoopLORA(current_count,battery_voltage);
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
