#include <Arduino.h>        // Must come first — pins_arduino.h must be processed before heltec_unofficial.h defines its pin macros
#include <heltec_unofficial.h>
#include <LoRaWAN_ESP32.h>
#include "customs.h"
#include "pax.h"
#include "logging.h"
#include "ota.h"

LoRaWANNode* node;

void goToSleep() {
    if (node) persist.saveSession(node);
    uint32_t interval = node ? node->timeUntilUplink() : 0;
    uint32_t sleepSec = max(interval / 1000, (uint32_t)SLEEP_TIME_SEC);
    logMessage("Sleeping for " + String(sleepSec) + "s");
    heltec_deep_sleep(sleepSec);
}

void setup() {
    heltec_setup();

    // ── 0. OTA check ─────────────────────────────────────────────
    if (shouldEnterOtaMode()) {
        logMessage("Entering WiFi OTA mode...");
        performWifiOta();
        // If we get here, OTA failed — continue normal operation
    }

    // ── 1. BLE scan ──────────────────────────────────────────────
    paxSetup();
    unsigned long scanStart = millis();
    while (!new_data_available && millis() - scanStart < 35000) {
        delay(100);
    }
    uint16_t paxCount = current_count;
    paxStop();
    logMessageF("PAX count: %d", paxCount);

    // ── 2. Battery ───────────────────────────────────────────────
    float voltage = heltec_vbat();
    float battPct = heltec_battery_percent(voltage);
    logMessageF("Battery: %.2fV (%.1f%%)", voltage, battPct);

    // ── 3. Radio init ────────────────────────────────────────────
    if (radio.begin() != RADIOLIB_ERR_NONE) {
        logMessage("Radio init failed");
        goToSleep();
    }

    // ── 4. Provision + join ──────────────────────────────────────
    if (!persist.isProvisioned()) {
        persist.provision("EU868", 0, JOINEUI, DEVEUI,
                          (uint8_t*)APPKEY, (uint8_t*)APPKEY);
    }
    node = persist.manage(&radio);
    if (!node->isActivated()) {
        logMessage("Join failed");
        goToSleep();
    }

    // ── 5. Build payload ─────────────────────────────────────────
    char payload[96];
    snprintf(payload, sizeof(payload), "%s,0,%.4f,1,%.4f",
             sensor_id, paxCount * factor, battPct);
    logMessage("Payload: " + String(payload));

    // ── 6. Send ──────────────────────────────────────────────────
    uint8_t downlink[256];
    size_t downlinkLen = sizeof(downlink);
    int16_t state = node->sendReceive(
        (uint8_t*)payload, strlen(payload), 2, downlink, &downlinkLen);
    if (state >= 0) {
        logMessage("TX ok");
    } else {
        logMessageF("TX error %d", state);
    }

    // ── 7. Downlink: OTA trigger ──────────────────────────────────
    if (downlinkLen > 0 && downlink[0] == 0x01) {
        logMessage("OTA requested via downlink");
        setOtaFlag();
        // Flag is set; OTA will run on next wake
    }

    goToSleep();
}

void loop() {}
