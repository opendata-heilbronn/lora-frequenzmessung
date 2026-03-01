#pragma once
#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <HTTPClient.h>
#include <Update.h>

#define CODEBERG_BASE "https://codeberg.org/cfhn/lora-frequenzmessung/releases/download/"
#define OTA_ASSET     "/heltec_wifi_lora_32_V3_firmware.bin"
#define OTA_WIFI_TIMEOUT_MS (30UL * 60 * 1000)  // 30 minutes

// Connects to WiFi, downloads firmware from Codeberg releases, flashes OTA-B partition.
// On success calls ESP.restart() — never returns. Returns false on any failure.
bool performWiFiOTA(const char* ssid, const char* password, const char* tag) {
    // 1. Connect WiFi with 30 min timeout
    WiFi.begin(ssid, password);
    unsigned long start = millis();
    while (WiFi.status() != WL_CONNECTED) {
        if (millis() - start > OTA_WIFI_TIMEOUT_MS) {
            WiFi.disconnect(true, true);
            Serial.println("OTA: WiFi timeout after 30 min");
            return false;
        }
        delay(500);
    }
    Serial.println("OTA: WiFi connected");

    // 2. Build URL and fetch
    char url[192];
    snprintf(url, sizeof(url), "%s%s%s", CODEBERG_BASE, tag, OTA_ASSET);
    Serial.printf("OTA: fetching %s\n", url);

    WiFiClientSecure client;
    client.setInsecure();  // Codeberg uses valid TLS; skip cert pin for simplicity
    HTTPClient http;
    http.begin(client, url);
    http.setFollowRedirects(HTTPC_STRICT_FOLLOW_REDIRECTS);
    int code = http.GET();
    if (code != HTTP_CODE_OK) {
        Serial.printf("OTA: HTTP error %d\n", code);
        http.end();
        WiFi.disconnect(true, true);
        return false;
    }

    // 3. Stream into OTA-B partition
    int totalLen = http.getSize();
    WiFiClient* stream = http.getStreamPtr();
    if (!Update.begin(totalLen)) {
        Serial.println("OTA: Update.begin failed");
        http.end();
        WiFi.disconnect(true, true);
        return false;
    }
    size_t written = Update.writeStream(*stream);
    if ((int)written != totalLen || !Update.end(true)) {
        Serial.println("OTA: flash write failed");
        http.end();
        WiFi.disconnect(true, true);
        return false;
    }
    http.end();
    WiFi.disconnect(true, true);
    Serial.println("OTA: success, rebooting");
    ESP.restart();
    return true;  // never reached
}
