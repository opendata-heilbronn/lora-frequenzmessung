#ifndef CUSTOMS_H
#define CUSTOMS_H

#include <cstdint>

// Global logging flag: set to 1 to enable logging, 0 to disable
#define ENABLE_LOGGING 1

const char sensor_id[] = "f99a22e6";
const float factor     = 0.7;
const int SLEEP_TIME_SEC = 900;

// LoRaWAN OTAA credentials
const uint64_t JOINEUI = 0x0100000000000000ULL;
const uint64_t DEVEUI  = 0x70B3D57ED00736DFULL;
const uint8_t  APPKEY[16] = {0x2A, 0x24, 0xAC, 0x8B, 0x7F, 0x08, 0x41, 0x4B,
                              0xCE, 0x8A, 0x2E, 0x1A, 0xC8, 0xA8, 0xA2, 0x03};

// WiFi OTA settings
const char OTA_WIFI_SSID[] = "SensorPAX-Update";
const char OTA_WIFI_PASS[] = "changeme";
const char OTA_FIRMWARE_URL[] = "http://192.168.4.1/firmware.bin";
const int  OTA_TIMEOUT_SEC = 60;


#endif // CUSTOMS_H
