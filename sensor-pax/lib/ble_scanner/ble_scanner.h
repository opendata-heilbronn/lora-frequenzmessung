#ifndef _BLE_SCANNER_H
#define _BLE_SCANNER_H

#include <stdint.h>

/**
 * BLE Scanner Module
 *
 * Energy-efficient BLE scanner using NimBLE.
 * Designed for scan-on-demand architecture with deep sleep.
 *
 * Usage:
 *   1. ble_scanner_init(rssi_threshold) - Initialize scanner
 *   2. ble_scanner_scan(duration, &result) - Scan for devices
 *   3. ble_scanner_deinit() - Release resources before sleep
 */

// Scan result structure
typedef struct {
    uint16_t ble_count;     // Number of unique BLE devices detected
    uint32_t scan_time_ms;  // Actual scan duration in milliseconds
} ble_scan_result_t;

/**
 * Initialize the BLE scanner
 *
 * @param rssi_threshold Minimum RSSI to count devices (e.g., -80 dBm)
 * @return 0 on success, negative on error
 */
int ble_scanner_init(int rssi_threshold);

/**
 * Perform a BLE scan (blocking)
 *
 * @param duration_sec Scan duration in seconds
 * @param result Pointer to result structure (filled on return)
 * @return 0 on success, negative on error
 */
int ble_scanner_scan(uint32_t duration_sec, ble_scan_result_t* result);

/**
 * Deinitialize BLE scanner before deep sleep
 * Releases all BLE resources to allow proper deep sleep
 */
void ble_scanner_deinit(void);

#endif // _BLE_SCANNER_H
