#pragma once
#include <Arduino.h>

// Downlink command IDs (first byte of downlink payload)
#define CMD_REQUEST_VERSION 0x01
#define CMD_OTA_UPDATE      0x02

// Persists through deep sleep — set when version report is requested via downlink
RTC_DATA_ATTR bool reportVersion = true;

// OTA flags — set by CMD_OTA_UPDATE downlink, acted on in next wake cycle
RTC_DATA_ATTR bool doOTA = false;
RTC_DATA_ATTR char otaTag[32] = "";  // e.g. "firmware-pax-v1.3"

// Call after sendReceive() to process any received downlink.
// Sets flags that are acted on in the next uplink payload.
inline void handleDownlink(uint8_t* buf, size_t len) {
    if (len == 0) {
        Serial.println("Downlink: none");
        return;
    }
    Serial.printf("Downlink: %d bytes, cmd=0x%02X\n", (int)len, buf[0]);
    switch (buf[0]) {
        case CMD_REQUEST_VERSION:
            reportVersion = true;
            Serial.println("Downlink: CMD_REQUEST_VERSION → reportVersion=true");
            break;
        case CMD_OTA_UPDATE:
            if (len > 1 && len - 1 < sizeof(otaTag)) {
                memcpy(otaTag, buf + 1, len - 1);
                otaTag[len - 1] = '\0';
                doOTA = true;
                Serial.printf("Downlink: CMD_OTA_UPDATE tag=%s\n", otaTag);
            } else {
                Serial.println("Downlink: CMD_OTA_UPDATE invalid payload");
            }
            break;
        default:
            Serial.printf("Downlink: unknown cmd 0x%02X\n", buf[0]);
            break;
    }
}
