#ifndef PAX_H
#define PAX_H

#include "ble_scanner.h"
#include "customs.h"
#include "logging.h"

// Global scan result available after scanning
ble_scan_result_t last_scan_result;

// Initialize the BLE scanner
void InitPAX() {
    logMessage("Initializing BLE scanner...");
    int result = ble_scanner_init(BLE_RSSI_THRESHOLD);
    if (result != 0) {
        logMessage("BLE scanner init failed: " + String(result));
    } else {
        logMessage("BLE scanner initialized (RSSI threshold: " + String(BLE_RSSI_THRESHOLD) + " dBm)");
    }
}

// Perform a BLE scan and return the count
int ScanPAX() {
    logMessage("Starting BLE scan for " + String(BLE_SCAN_DURATION_SEC) + " seconds...");

    int result = ble_scanner_scan(BLE_SCAN_DURATION_SEC, &last_scan_result);

    if (result != 0) {
        logMessage("BLE scan failed: " + String(result));
        return 0;
    }

    logMessage("Scan complete: " + String(last_scan_result.ble_count) +
               " devices in " + String(last_scan_result.scan_time_ms) + "ms");

    return last_scan_result.ble_count;
}

// Deinitialize BLE scanner before deep sleep
void DeinitPAX() {
    logMessage("Deinitializing BLE scanner...");
    ble_scanner_deinit();
}

#endif // PAX_H
