#ifndef PROVISION_H
#define PROVISION_H

#include <Arduino.h>
#include <Preferences.h>
#include <LoRaWAN_ESP32.h>

#define APP_NVS_NAMESPACE  "sensorpax"
#define KEY_SENSOR_ID      "sensor_id"   // ≤15 chars NVS key
#define KEY_FACTOR         "factor"
#define KEY_SLEEP_TIME     "sleep_sec"

struct Config {
    char    sensor_id[16];
    float   factor;
    int32_t sleep_time_sec;
};

// Convert a 16-char hex string to uint64_t. Returns false on invalid input.
static bool hexToUint64(const char* hex, uint64_t* out) {
    if (strlen(hex) != 16) return false;
    *out = 0;
    for (int i = 0; i < 16; i++) {
        char c = hex[i];
        uint8_t nibble;
        if      (c >= '0' && c <= '9') nibble = c - '0';
        else if (c >= 'a' && c <= 'f') nibble = c - 'a' + 10;
        else if (c >= 'A' && c <= 'F') nibble = c - 'A' + 10;
        else return false;
        *out = (*out << 4) | nibble;
    }
    return true;
}

// Convert a (len*2)-char hex string into len bytes. Returns false on invalid input.
static bool hexToBytes(const char* hex, uint8_t* buf, size_t len) {
    if (strlen(hex) != len * 2) return false;
    for (size_t i = 0; i < len; i++) {
        char hi = hex[i * 2], lo = hex[i * 2 + 1];
        uint8_t hi_n, lo_n;
        if      (hi >= '0' && hi <= '9') hi_n = hi - '0';
        else if (hi >= 'a' && hi <= 'f') hi_n = hi - 'a' + 10;
        else if (hi >= 'A' && hi <= 'F') hi_n = hi - 'A' + 10;
        else return false;
        if      (lo >= '0' && lo <= '9') lo_n = lo - '0';
        else if (lo >= 'a' && lo <= 'f') lo_n = lo - 'a' + 10;
        else if (lo >= 'A' && lo <= 'F') lo_n = lo - 'A' + 10;
        else return false;
        buf[i] = (hi_n << 4) | lo_n;
    }
    return true;
}

// Returns true when both LoRaWAN NVS ("lorawan") and app NVS ("sensorpax") are fully provisioned.
// Side effect: calling persist.isProvisioned() loads LoRaWAN state into the persist object.
inline bool isConfigProvisioned() {
    if (!persist.isProvisioned()) return false;
    Preferences p;
    p.begin(APP_NVS_NAMESPACE, true);
    bool ok = p.isKey(KEY_SENSOR_ID) && p.isKey(KEY_FACTOR) && p.isKey(KEY_SLEEP_TIME);
    p.end();
    return ok;
}

// Load app config from NVS. Call only after isConfigProvisioned() returns true.
inline Config loadConfig() {
    Config cfg;
    Preferences p;
    p.begin(APP_NVS_NAMESPACE, true);
    p.getString(KEY_SENSOR_ID, cfg.sensor_id, sizeof(cfg.sensor_id));
    cfg.factor         = p.getFloat(KEY_FACTOR, 0.7f);
    cfg.sleep_time_sec = p.getInt(KEY_SLEEP_TIME, 900);
    p.end();
    return cfg;
}

// Block until a valid provisioning session completes over Serial.
// Protocol: send KEY=VALUE lines, device replies OK or ERROR: ..., then send COMMIT.
// On COMMIT the device writes NVS and calls ESP.restart().
// Supported keys: sensor_id, factor, sleep_sec, joineui (16 hex), deveui (16 hex), appkey (32 hex)
inline void waitForProvisioning() {
    Serial.println("PROV_READY");
    Serial.println("Send KEY=VALUE lines then COMMIT");
    unsigned long lastAnnounce = millis();

    // Staged values
    char    v_sensor_id[16] = {};
    char    v_joineui[17]   = {};
    char    v_deveui[17]    = {};
    char    v_appkey[33]    = {};
    float   v_factor        = 0.0f;
    int32_t v_sleep_sec     = 0;
    bool    has_sensor_id   = false;
    bool    has_joineui     = false;
    bool    has_deveui      = false;
    bool    has_appkey      = false;
    bool    has_factor      = false;
    bool    has_sleep_sec   = false;

    String line = "";
    while (true) {
        if (!Serial.available()) {
            // Re-announce every 2s so scripts connecting after boot still see PROV_READY
            if (millis() - lastAnnounce > 2000) {
                Serial.println("PROV_READY");
                lastAnnounce = millis();
            }
            continue;
        }
        lastAnnounce = millis();  // reset timer on any activity
        char c = Serial.read();
        if (c == '\r') continue;
        if (c != '\n') {
            line += c;
            continue;
        }

        // Process completed line
        line.trim();
        if (line.length() == 0) {
            line = "";
            continue;
        }

        if (line == "COMMIT") {
            if (!has_sensor_id || !has_joineui || !has_deveui ||
                !has_appkey || !has_factor || !has_sleep_sec) {
                Serial.println("ERROR: missing keys before COMMIT");
            } else {
                // Write app namespace
                Preferences prefs;
                prefs.begin(APP_NVS_NAMESPACE, false);
                prefs.putString(KEY_SENSOR_ID, v_sensor_id);
                prefs.putFloat(KEY_FACTOR, v_factor);
                prefs.putInt(KEY_SLEEP_TIME, v_sleep_sec);
                prefs.end();

                // Write LoRaWAN credentials
                uint64_t joineui, deveui;
                uint8_t  appkey[16];
                hexToUint64(v_joineui, &joineui);
                hexToUint64(v_deveui,  &deveui);
                hexToBytes(v_appkey, appkey, 16);
                persist.provision("EU868", 0, joineui, deveui, appkey, appkey);

                Serial.println("OK");
                Serial.flush();
                delay(100);
                ESP.restart();
            }
            line = "";
            continue;
        }

        int eqIdx = line.indexOf('=');
        if (eqIdx < 1) {
            Serial.println("ERROR: bad format, expected KEY=VALUE");
            line = "";
            continue;
        }

        String key = line.substring(0, eqIdx);
        String val = line.substring(eqIdx + 1);

        if (key == "sensor_id") {
            if (val.length() == 0 || val.length() >= sizeof(v_sensor_id)) {
                Serial.println("ERROR: sensor_id must be 1–15 chars");
            } else {
                strncpy(v_sensor_id, val.c_str(), sizeof(v_sensor_id) - 1);
                has_sensor_id = true;
                Serial.println("OK");
            }
        } else if (key == "factor") {
            v_factor   = val.toFloat();
            has_factor = true;
            Serial.println("OK");
        } else if (key == "sleep_sec") {
            v_sleep_sec   = val.toInt();
            has_sleep_sec = true;
            Serial.println("OK");
        } else if (key == "joineui") {
            if (val.length() != 16) {
                Serial.println("ERROR: joineui must be 16 hex chars");
            } else {
                strncpy(v_joineui, val.c_str(), sizeof(v_joineui) - 1);
                has_joineui = true;
                Serial.println("OK");
            }
        } else if (key == "deveui") {
            if (val.length() != 16) {
                Serial.println("ERROR: deveui must be 16 hex chars");
            } else {
                strncpy(v_deveui, val.c_str(), sizeof(v_deveui) - 1);
                has_deveui = true;
                Serial.println("OK");
            }
        } else if (key == "appkey") {
            if (val.length() != 32) {
                Serial.println("ERROR: appkey must be 32 hex chars");
            } else {
                strncpy(v_appkey, val.c_str(), sizeof(v_appkey) - 1);
                has_appkey = true;
                Serial.println("OK");
            }
        } else {
            Serial.println("ERROR: unknown key");
        }

        line = "";
    }
}

#endif // PROVISION_H
