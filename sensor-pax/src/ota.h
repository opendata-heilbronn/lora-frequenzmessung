#ifndef OTA_H
#define OTA_H

#include <Preferences.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include <Update.h>
#include "customs.h"
#include "logging.h"

// PRG button pin on Heltec WiFi LoRa 32 V3
#ifndef PRG_BUTTON_PIN
#define PRG_BUTTON_PIN 0
#endif

static const char* OTA_NVS_NS  = "ota";
static const char* OTA_NVS_KEY = "pending";

void setOtaFlag() {
    Preferences prefs;
    prefs.begin(OTA_NVS_NS, false);
    prefs.putBool(OTA_NVS_KEY, true);
    prefs.end();
}

void clearOtaFlag() {
    Preferences prefs;
    prefs.begin(OTA_NVS_NS, false);
    prefs.putBool(OTA_NVS_KEY, false);
    prefs.end();
}

bool shouldEnterOtaMode() {
    // PRG button held low at boot
    pinMode(PRG_BUTTON_PIN, INPUT_PULLUP);
    if (digitalRead(PRG_BUTTON_PIN) == LOW) {
        return true;
    }
    // NVS flag set by downlink
    Preferences prefs;
    prefs.begin(OTA_NVS_NS, true);
    bool flag = prefs.getBool(OTA_NVS_KEY, false);
    prefs.end();
    return flag;
}

void performWifiOta() {
    logMessage("WiFi OTA: connecting to " + String(OTA_WIFI_SSID));
    WiFi.begin(OTA_WIFI_SSID, OTA_WIFI_PASS);

    unsigned long wifiStart = millis();
    while (WiFi.status() != WL_CONNECTED) {
        if (millis() - wifiStart > 15000UL) {
            logMessage("WiFi OTA: connection timeout");
            WiFi.disconnect(true);
            clearOtaFlag();
            return;
        }
        delay(200);
    }
    logMessage("WiFi OTA: connected, IP=" + WiFi.localIP().toString());

    HTTPClient http;
    http.begin(OTA_FIRMWARE_URL);
    http.setTimeout(OTA_TIMEOUT_SEC * 1000);

    int httpCode = http.GET();
    if (httpCode != HTTP_CODE_OK) {
        logMessageF("WiFi OTA: HTTP error %d", httpCode);
        http.end();
        WiFi.disconnect(true);
        clearOtaFlag();
        return;
    }

    int contentLen = http.getSize();
    logMessageF("WiFi OTA: firmware size %d bytes", contentLen);

    if (!Update.begin(contentLen > 0 ? contentLen : UPDATE_SIZE_UNKNOWN)) {
        logMessageF("WiFi OTA: Update.begin failed, error=%d", Update.getError());
        http.end();
        WiFi.disconnect(true);
        clearOtaFlag();
        return;
    }

    WiFiClient* stream = http.getStreamPtr();
    size_t written = Update.writeStream(*stream);
    logMessageF("WiFi OTA: wrote %u bytes", (unsigned)written);

    if (!Update.end(true)) {
        logMessageF("WiFi OTA: Update.end failed, error=%d", Update.getError());
        http.end();
        WiFi.disconnect(true);
        clearOtaFlag();
        return;
    }

    http.end();
    WiFi.disconnect(true);
    clearOtaFlag();
    logMessage("WiFi OTA: success, rebooting...");
    delay(200);
    ESP.restart();
}

#endif // OTA_H
