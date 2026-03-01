#pragma once
#include <Arduino.h>

// Downlink command IDs (first byte of downlink payload)
#define CMD_REQUEST_VERSION 0x01
// Future: #define CMD_OTA_UPDATE 0x02

// Persists through deep sleep — set when version report is requested via downlink
RTC_DATA_ATTR bool reportVersion = true;

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
        default:
            Serial.printf("Downlink: unknown cmd 0x%02X\n", buf[0]);
            break;
    }
}
