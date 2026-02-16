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
extern char sensor_id[];

extern uint8_t devEui[];
extern uint8_t appEui[];
extern uint8_t appKey[];

/* ABP para*/
extern uint8_t nwkSKey[];
extern uint8_t appSKey[];
extern uint32_t devAddr;
extern uint16_t userChannelsMask[6];

// PAX count factor (multiply raw count by this to estimate actual people)
extern float factor;

// Sleep time in seconds (900s = 15 min total cycle)
// With 30s scan, actual sleep is ~870s (14.5 min)
extern float sleepTime;

extern int SensorTypFrequency;
extern int SensorTypBattery;

#endif // CUSTOMS_H
