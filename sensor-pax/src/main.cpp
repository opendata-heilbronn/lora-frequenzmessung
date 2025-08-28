#define LoRaWAN_DEBUG_LEVEL 0

#define uS_TO_S_FACTOR 1000000ULL
#define TIME_TO_SLEEP 60 // 900 Sec are 15 minutes

#include <Arduino.h>
#include "HT_lCMEN2R13EFC1.h"

#include "lora.h"
#include "pax.h"


void setup()
{
  Serial.begin(115200);
  delay(1000); // Take some time to open up the Serial Monitor

  esp_sleep_enable_timer_wakeup(TIME_TO_SLEEP * uS_TO_S_FACTOR);
  Serial.println("Setup ESP32 to sleep for every " + String(TIME_TO_SLEEP) + " Seconds");

  InitPAX();
  InitLORA();
}

void loop()
{
  LoopLORA(current_count);
}