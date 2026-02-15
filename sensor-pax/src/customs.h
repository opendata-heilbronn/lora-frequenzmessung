#ifndef CUSTOMS_H
#define CUSTOMS_H

#include <cstdint>

// Global logging flag: set to 1 to enable logging, 0 to disable
// NOTE: Disabling logging (set to 0) saves additional power in production by reducing Serial usage
#define ENABLE_LOGGING 1

// Global display flag: set to 1 to enable display functions, 0 to disable
// NOTE: Disabling display (set to 0) saves additional power by keeping the display off
#define ENABLE_DISPLAY 0

// --- BLE Scanner Configuration ---
// RSSI threshold: only count devices with signal stronger than this (-80 dBm typical)
#define BLE_RSSI_THRESHOLD -80

// Scan duration in seconds (30s provides good accuracy)
#define BLE_SCAN_DURATION_SEC 30

// Minimum PAX count to transmit (privacy: don't send if less than 6 people)
#define MIN_PAX_TO_SEND 6

// --- Device Identification ---
char sensor_id[] = "863f75b0";

uint8_t devEui[] = {0x70, 0xB3, 0xD5, 0x7E, 0xD0, 0x07, 0x2D, 0x3C};
uint8_t appEui[] = {0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01};
uint8_t appKey[] = {0xC3, 0xD6, 0x99, 0x69, 0x1F, 0x7B, 0x58, 0x9A, 0x83, 0xEC, 0x4E, 0x28, 0x85, 0xAF, 0xE9, 0x62};

/* ABP para*/
uint8_t nwkSKey[] = {0x08, 0xF6, 0xA5, 0x8E, 0x20, 0xE3, 0xAC, 0x24, 0x95, 0x1F, 0xFD, 0xCE, 0x03, 0x7E, 0x9A, 0x48};
uint8_t appSKey[] = {0x12, 0x08, 0x61, 0xE5, 0x38, 0x60, 0x56, 0xE6, 0xC1, 0xE9, 0x6B, 0x2C, 0x97, 0xC4, 0x2E, 0xEE};
uint32_t devAddr = (uint32_t)0x260BE3E9;
uint16_t userChannelsMask[6] = {0x00FF, 0x0000, 0x0000, 0x0000, 0x0000, 0x0000};

// PAX count factor (multiply raw count by this to estimate actual people)
float factor= 0.7;

// Sleep time in seconds (900s = 15 min total cycle)
// With 30s scan, actual sleep is ~870s (14.5 min)
float sleepTime= 900;

int SensorTypFrequency=0;
int SensorTypBattery=1;

#endif // CUSTOMS_H
