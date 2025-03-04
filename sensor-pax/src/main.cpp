#define LIBPAX_ARDUINO 1
#define LIBPAX_WIFI 0
#define LIBPAX_BLE 1

#include <Arduino.h>
#include "esp_wifi.h"
#include "libpax_api.h"


#define uS_TO_S_FACTOR 1000000ULL /* Conversion factor for micro seconds to seconds */
#define TIME_TO_SLEEP 120          /* Time ESP32 will go to sleep (in seconds) */

RTC_DATA_ATTR int bootCount = 0;

count_payload_t* current_count;

wifi_country_t country;

void print_wakeup_reason()
{
  esp_sleep_wakeup_cause_t wakeup_reason;

  wakeup_reason = esp_sleep_get_wakeup_cause();

  switch (wakeup_reason)
  {
  case ESP_SLEEP_WAKEUP_EXT0:
    Serial.println("Wakeup caused by external signal using RTC_IO");
    break;
  case ESP_SLEEP_WAKEUP_EXT1:
    Serial.println("Wakeup caused by external signal using RTC_CNTL");
    break;
  case ESP_SLEEP_WAKEUP_TIMER:
    Serial.println("Wakeup caused by timer");
    break;
  case ESP_SLEEP_WAKEUP_TOUCHPAD:
    Serial.println("Wakeup caused by touchpad");
    break;
  case ESP_SLEEP_WAKEUP_ULP:
    Serial.println("Wakeup caused by ULP program");
    break;
  default:
    Serial.printf("Wakeup was not caused by deep sleep: %d\n", wakeup_reason);
    break;
  }
}

void pax_callback(void)
{
  Serial.println("PAX callback");

  Serial.println("PAX: " + String(current_count->pax));
//  Serial.println("WiFi: " + String(current_count->wifi_count));
  Serial.println("BLE: " + String(current_count->ble_count));

  // start deep sleep
  esp_sleep_enable_timer_wakeup(TIME_TO_SLEEP * uS_TO_S_FACTOR);
  Serial.println("Setup ESP32 to sleep for every " + String(TIME_TO_SLEEP) + " Seconds");

  Serial.println("Going to sleep now");
  Serial.flush();
  esp_deep_sleep_start();
}

void setup()
{
  Serial.begin(115200);
  delay(1000); // Take some time to open up the Serial Monitor

  // Increment boot number and print it every reboot
  ++bootCount;
  Serial.println("Boot number: " + String(bootCount));

  // Print the wakeup reason for ESP32
  print_wakeup_reason();

  libpax_config_t* config = (libpax_config_t*)malloc(sizeof(libpax_config_t));

  Serial.println("PAX config init");
  libpax_default_config(config);

  config->wificounter = LIBPAX_WIFI;

  Serial.println("updating pax config");
  libpax_update_config(config);

  Serial.println("PAX counter init");
  libpax_counter_init(pax_callback, current_count, 20, 0);
  
  delay(1000);
}

bool started = false;

void loop()
{
  if (!started)
  {
    Serial.println("Starting PAX counter");
    libpax_counter_start();
    started = true;
  }
  delay(100);
}