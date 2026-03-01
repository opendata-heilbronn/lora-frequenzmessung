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
    if (len == 0) return;
    switch (buf[0]) {
        case CMD_REQUEST_VERSION:
            reportVersion = true;
            break;
        default:
            break;
    }
}
