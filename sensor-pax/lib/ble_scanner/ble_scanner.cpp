/*
 * BLE Scanner Implementation
 *
 * Energy-efficient NimBLE-based scanner for PAX counting.
 * Supports scan-on-demand with proper deep sleep integration.
 */

#include "ble_scanner.h"
#include "mac_counter.h"
#include <Arduino.h>
#include <NimBLEDevice.h>

// Scanner configuration
static int g_rssi_threshold = -80;
static bool g_initialized = false;

// Scan parameters for power efficiency
// Scan window: 50ms, Scan interval: 100ms = 50% duty cycle
static const uint16_t SCAN_WINDOW_MS = 50;
static const uint16_t SCAN_INTERVAL_MS = 100;

// Callback class for BLE advertisements
class ScanCallbacks : public NimBLEAdvertisedDeviceCallbacks {
public:
    void onResult(NimBLEAdvertisedDevice* advertisedDevice) override {
        // Get MAC address
        NimBLEAddress addr = advertisedDevice->getAddress();
        const uint8_t* mac = addr.getNative();

        // Get RSSI
        int rssi = advertisedDevice->getRSSI();

        // Add to counter (filters by RSSI and MAC type internally)
        mac_counter_add(mac, rssi, g_rssi_threshold);
    }
};

static ScanCallbacks scanCallbacks;

int ble_scanner_init(int rssi_threshold) {
    if (g_initialized) {
        return 0;  // Already initialized
    }

    g_rssi_threshold = rssi_threshold;

    // Initialize NimBLE
    NimBLEDevice::init("");

    // Power optimization: use minimum TX power (we're only receiving)
    NimBLEDevice::setPower(ESP_PWR_LVL_N12);

    g_initialized = true;
    return 0;
}

int ble_scanner_scan(uint32_t duration_sec, ble_scan_result_t* result) {
    if (!g_initialized) {
        return -1;
    }

    if (result == nullptr) {
        return -2;
    }

    // Reset MAC counter for new scan
    mac_counter_reset();

    // Get scanner instance
    NimBLEScan* pScan = NimBLEDevice::getScan();

    // Configure scanner
    pScan->setAdvertisedDeviceCallbacks(&scanCallbacks, true);  // true = want duplicates (we handle dedup)
    pScan->setActiveScan(false);  // Passive scan - no TX, saves power
    pScan->setInterval(SCAN_INTERVAL_MS);
    pScan->setWindow(SCAN_WINDOW_MS);
    pScan->setDuplicateFilter(false);  // We handle dedup in mac_counter
    pScan->setMaxResults(0);  // Don't store results, just callback

    // Record start time
    uint32_t start_ms = millis();

    // Start blocking scan (duration in seconds)
    pScan->start(duration_sec, false);  // false = blocking

    // Fill result
    result->ble_count = mac_counter_get_count();
    result->scan_time_ms = millis() - start_ms;

    // Clear scan results to free memory
    pScan->clearResults();

    return 0;
}

void ble_scanner_deinit(void) {
    if (!g_initialized) {
        return;
    }

    // Stop any ongoing scan
    NimBLEScan* pScan = NimBLEDevice::getScan();
    if (pScan->isScanning()) {
        pScan->stop();
    }

    // Deinitialize NimBLE to release Bluetooth resources
    NimBLEDevice::deinit(true);  // true = release memory

    g_initialized = false;
}
