#include <Arduino.h>        // Must come first — pins_arduino.h must be processed before heltec_unofficial.h defines its pin macros
#include <heltec_unofficial.h>
#include <LoRaWAN_ESP32.h>
#include "provision.h"
#include "pax.h"
#include "logging.h"
#include "version.h"
#include "downlink.h"

LoRaWANNode* node;
Config cfg;

void goToSleep() {
    if (node) persist.saveSession(node);
    uint32_t interval = node ? node->timeUntilUplink() : 0;
    uint32_t sleepSec = max(interval / 1000, (uint32_t)cfg.sleep_time_sec);
    logMessageF("Sleeping for %us", sleepSec);
    heltec_deep_sleep(sleepSec);
}

void setup() {
    heltec_setup();

    if (!isConfigProvisioned()) {
        waitForProvisioning();
        while (true) delay(1000);  // reboots on success; never reached
    }
    cfg = loadConfig();

    // ── 1. Battery (read BEFORE BLE to avoid radio interference) ─
    pinMode(VBAT_CTRL, OUTPUT);
    digitalWrite(VBAT_CTRL, HIGH);  // HIGH enables the N-FET on this board
    delay(10);
    (void)analogReadMilliVolts(VBAT_ADC);  // dummy read for ADC stabilisation
    delay(2);
    float voltage = analogReadMilliVolts(VBAT_ADC) / 1000.0f * 4.9f;
    pinMode(VBAT_CTRL, INPUT);
    float battPct = heltec_battery_percent(voltage);
    logMessageF("Battery: %.2fV (%.1f%%)", voltage, battPct);

    // ── 2. BLE scan ──────────────────────────────────────────────
    paxSetup();
    unsigned long scanStart = millis();
    while (!new_data_available && millis() - scanStart < 35000) {
        delay(100);
    }
    uint16_t paxCount = current_count;
    paxStop();
    logMessageF("PAX count: %d", paxCount);

    // ── 3. Radio init ────────────────────────────────────────────
    if (radio.begin() != RADIOLIB_ERR_NONE) {
        logMessage("Radio init failed");
        goToSleep();
    }

    // ── 4. Join ──────────────────────────────────────────────────
    node = persist.manage(&radio);
    if (!node->isActivated()) {
        logMessage("Join failed");
        goToSleep();
    }

    // ── 5. Build payload ─────────────────────────────────────────
    char payload[128];
    int payloadLen = snprintf(payload, sizeof(payload), "%s,0,%.4f,1,%.4f",
             cfg.sensor_id, paxCount * cfg.factor, battPct);
    if (reportVersion) {
        payloadLen += snprintf(payload + payloadLen, sizeof(payload) - payloadLen,
                               ",2,%s", FIRMWARE_VERSION);
        reportVersion = false;
    }
    logMessageF("Payload: %s", payload);

    // ── 6. Send ──────────────────────────────────────────────────
    uint8_t downlink[256];
    size_t downlinkLen = sizeof(downlink);
    int16_t state = node->sendReceive(
        (uint8_t*)payload, strlen(payload), 2, downlink, &downlinkLen);
    logMessageF("sendReceive state=%d downlinkLen=%d", state, (int)downlinkLen);
    if (state >= 0) {
        logMessage("TX ok");
        handleDownlink(downlink, downlinkLen);
    } else {
        logMessageF("TX error %d", state);
    }

    goToSleep();
}

void loop() {}
