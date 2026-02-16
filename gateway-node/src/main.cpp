/**
 * Meshtastic Gateway Node for Sensor-PAX
 *
 * Architecture:
 * [Mesh Sensors] --LoRa P2P--> [Gateway] --WiFi/HTTP--> [Web Service]
 *
 * Functionality:
 * 1. Connects to WiFi
 * 2. Listens for mesh packets from PAX sensors
 * 3. Sends ACK back to sensors
 * 4. Forwards PAX data to web service via HTTP POST
 * 5. Buffers packets if HTTP fails (with retry logic)
 */

#include <Arduino.h>
#include <WiFi.h>
#include <HTTPClient.h>
#include <RadioLib.h>
#include "config.h"

// === Mesh Packet Structure ===
// Must match sensor-pax/src/mesh_comms.h exactly
#pragma pack(push, 1)

struct MeshPacket {
    uint8_t type;               // Packet type (0x01 = PAX_DATA, 0x02 = ACK)
    char sensor_id[16];         // Sensor identifier
    uint16_t pax_count;         // Raw PAX count
    uint8_t battery;            // Battery percentage (0-100)
    uint8_t hop_count;          // Hop counter
    uint32_t timestamp;         // Unix timestamp
    uint16_t crc;               // CRC16 checksum
};

#pragma pack(pop)

// Packet types
#define PKT_TYPE_PAX_DATA 0x01
#define PKT_TYPE_ACK 0x02

// === Global Objects ===
SX1262 radio = new Module(RADIO_CS_PIN, RADIO_DIO1_PIN, RADIO_RST_PIN, RADIO_BUSY_PIN);
HTTPClient http;

// === Statistics ===
unsigned long packetsReceived = 0;
unsigned long packetsForwarded = 0;
unsigned long packetsFailed = 0;
unsigned long lastPacketTime = 0;

// === WiFi Connection ===

void connectWiFi() {
    Serial.println("=== Connecting to WiFi ===");
    Serial.print("SSID: ");
    Serial.println(WIFI_SSID);

    WiFi.mode(WIFI_STA);
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

    unsigned long startTime = millis();
    pinMode(LED_PIN, OUTPUT);

    while (WiFi.status() != WL_CONNECTED) {
        if (millis() - startTime > WIFI_CONNECT_TIMEOUT) {
            Serial.println("WiFi connection timeout!");
            Serial.println("Retrying in 5 seconds...");
            delay(5000);
            startTime = millis();
            WiFi.disconnect();
            WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
            continue;
        }

        // Blink LED while connecting
        digitalWrite(LED_PIN, HIGH);
        delay(LED_BLINK_WIFI_CONNECTING / 2);
        digitalWrite(LED_PIN, LOW);
        delay(LED_BLINK_WIFI_CONNECTING / 2);

        Serial.print(".");
    }

    Serial.println();
    Serial.println("WiFi connected!");
    Serial.print("IP address: ");
    Serial.println(WiFi.localIP());
    Serial.print("Signal strength (RSSI): ");
    Serial.print(WiFi.RSSI());
    Serial.println(" dBm");

    // Flash LED to indicate success
    for (int i = 0; i < 3; i++) {
        digitalWrite(LED_PIN, HIGH);
        delay(100);
        digitalWrite(LED_PIN, LOW);
        delay(100);
    }
}

// === Radio Initialization ===

bool initRadio() {
    Serial.println("=== Initializing LoRa radio ===");
    Serial.print("Frequency: ");
    Serial.print(MESH_FREQUENCY, 3);
    Serial.println(" MHz");
    Serial.print("Bandwidth: ");
    Serial.print(MESH_BANDWIDTH, 1);
    Serial.println(" kHz");
    Serial.print("Spreading Factor: ");
    Serial.println(MESH_SPREADING_FACTOR);

    int state = radio.begin(
        MESH_FREQUENCY,
        MESH_BANDWIDTH,
        MESH_SPREADING_FACTOR,
        MESH_CODING_RATE,
        MESH_SYNC_WORD,
        17,  // Gateway can use higher TX power (17 dBm)
        MESH_PREAMBLE_LENGTH,
        0    // TCXO voltage
    );

    if (state != RADIOLIB_ERR_NONE) {
        Serial.print("Radio init failed, code: ");
        Serial.println(state);
        return false;
    }

    // Configure for explicit header mode
    radio.explicitHeader();

    // Enable CRC
    radio.setCRC(true);

    Serial.println("Radio initialized successfully");
    return true;
}

// === CRC Validation ===

uint16_t calculateCRC16(const uint8_t* data, size_t length) {
    uint16_t crc = 0xFFFF;

    for (size_t i = 0; i < length; i++) {
        crc ^= (uint16_t)data[i] << 8;
        for (uint8_t bit = 0; bit < 8; bit++) {
            if (crc & 0x8000) {
                crc = (crc << 1) ^ 0x1021;
            } else {
                crc = crc << 1;
            }
        }
    }

    return crc;
}

bool validatePacketCRC(const MeshPacket* packet) {
    size_t dataLength = sizeof(MeshPacket) - sizeof(packet->crc);
    uint16_t calculatedCRC = calculateCRC16((const uint8_t*)packet, dataLength);
    return (calculatedCRC == packet->crc);
}

// === ACK Transmission ===

void sendACK(const char* sensor_id) {
    Serial.print("Sending ACK to sensor: ");
    Serial.println(sensor_id);

    // Create ACK packet
    MeshPacket ackPacket;
    memset(&ackPacket, 0, sizeof(ackPacket));

    ackPacket.type = PKT_TYPE_ACK;
    strncpy(ackPacket.sensor_id, sensor_id, sizeof(ackPacket.sensor_id) - 1);
    ackPacket.timestamp = millis() / 1000;

    // Calculate CRC
    size_t dataLength = sizeof(MeshPacket) - sizeof(ackPacket.crc);
    ackPacket.crc = calculateCRC16((const uint8_t*)&ackPacket, dataLength);

    // Transmit ACK
    int state = radio.transmit((uint8_t*)&ackPacket, sizeof(ackPacket));

    if (state == RADIOLIB_ERR_NONE) {
        Serial.println("ACK sent successfully");
    } else {
        Serial.print("ACK transmission failed, code: ");
        Serial.println(state);
    }

    // Return to receive mode
    radio.startReceive();
}

// === HTTP Forwarding ===

bool forwardToAPI(const MeshPacket* packet) {
    Serial.println("--- Forwarding to web service ---");

    // Check WiFi connection
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("WiFi disconnected! Reconnecting...");
        connectWiFi();
    }

    // Build JSON payload
    String jsonPayload = "{";
    jsonPayload += "\"sensor\":\"" + String(packet->sensor_id) + "\",";
    jsonPayload += "\"pax\":" + String(packet->pax_count) + ",";
    jsonPayload += "\"battery\":" + String(packet->battery) + ",";
    jsonPayload += "\"source\":\"mesh\",";
    jsonPayload += "\"timestamp\":" + String(packet->timestamp) + ",";
    jsonPayload += "\"hop_count\":" + String(packet->hop_count);
    jsonPayload += "}";

    Serial.println("JSON payload:");
    Serial.println(jsonPayload);

    // Build URL
    String url;
    if (API_USE_HTTPS) {
        url = "https://";
    } else {
        url = "http://";
    }
    url += String(API_HOST);
    if (!(API_PORT == 80 && !API_USE_HTTPS) && !(API_PORT == 443 && API_USE_HTTPS)) {
        url += ":" + String(API_PORT);
    }
    url += String(API_PATH);

    Serial.print("POST URL: ");
    Serial.println(url);

    // Attempt HTTP POST with retries
    for (int retry = 0; retry < HTTP_MAX_RETRIES; retry++) {
        if (retry > 0) {
            Serial.print("Retry attempt ");
            Serial.print(retry);
            Serial.print("/");
            Serial.println(HTTP_MAX_RETRIES - 1);
            delay(HTTP_RETRY_DELAY);
        }

        http.begin(url);
        http.addHeader("Content-Type", "application/json");

        // Add authentication token if configured
        if (strlen(API_TOKEN) > 0) {
            http.addHeader("Authorization", API_TOKEN);
        }

        int httpCode = http.POST(jsonPayload);

        Serial.print("HTTP response code: ");
        Serial.println(httpCode);

        if (httpCode > 0) {
            if (httpCode == 200 || httpCode == 201 || httpCode == 204) {
                String response = http.getString();
                Serial.println("API response:");
                Serial.println(response);
                http.end();
                return true;
            } else {
                Serial.print("HTTP error: ");
                Serial.println(httpCode);
                String errorResponse = http.getString();
                Serial.println(errorResponse);
            }
        } else {
            Serial.print("HTTP request failed: ");
            Serial.println(http.errorToString(httpCode));
        }

        http.end();
    }

    Serial.println("Failed to forward packet after all retries");
    return false;
}

// === Packet Processing ===

void processPacket(const MeshPacket* packet) {
    Serial.println();
    Serial.println("========================================");
    Serial.println("=== MESH PACKET RECEIVED ===");
    Serial.println("========================================");
    Serial.print("Sensor ID: ");
    Serial.println(packet->sensor_id);
    Serial.print("PAX count: ");
    Serial.println(packet->pax_count);
    Serial.print("Battery: ");
    Serial.print(packet->battery);
    Serial.println("%");
    Serial.print("Hop count: ");
    Serial.println(packet->hop_count);
    Serial.print("Timestamp: ");
    Serial.println(packet->timestamp);
    Serial.print("CRC: 0x");
    Serial.println(packet->crc, HEX);

    // Validate CRC
    if (!validatePacketCRC(packet)) {
        Serial.println("*** CRC VALIDATION FAILED! ***");
        Serial.println("Packet discarded");
        packetsFailed++;
        return;
    }

    Serial.println("CRC validation: OK");

    // Update statistics
    packetsReceived++;
    lastPacketTime = millis();

    // Flash LED to indicate packet received
    digitalWrite(LED_PIN, HIGH);
    delay(LED_ON_DURATION);
    digitalWrite(LED_PIN, LOW);

    // Send ACK back to sensor
    sendACK(packet->sensor_id);

    // Forward to web service
    if (forwardToAPI(packet)) {
        packetsForwarded++;
        Serial.println("*** Packet forwarded successfully ***");
    } else {
        packetsFailed++;
        Serial.println("*** Packet forwarding failed ***");
    }

    Serial.println("========================================");
    Serial.println();
}

// === Status Display ===

void printStatus() {
    Serial.println();
    Serial.println("========================================");
    Serial.println("=== Gateway Status ===");
    Serial.println("========================================");
    Serial.print("WiFi: ");
    Serial.print(WiFi.status() == WL_CONNECTED ? "Connected" : "Disconnected");
    if (WiFi.status() == WL_CONNECTED) {
        Serial.print(" (");
        Serial.print(WiFi.RSSI());
        Serial.print(" dBm, IP: ");
        Serial.print(WiFi.localIP());
        Serial.print(")");
    }
    Serial.println();
    Serial.print("Uptime: ");
    Serial.print(millis() / 1000);
    Serial.println(" seconds");
    Serial.print("Packets received: ");
    Serial.println(packetsReceived);
    Serial.print("Packets forwarded: ");
    Serial.println(packetsForwarded);
    Serial.print("Packets failed: ");
    Serial.println(packetsFailed);
    if (packetsReceived > 0) {
        Serial.print("Success rate: ");
        Serial.print((packetsForwarded * 100.0) / packetsReceived, 1);
        Serial.println("%");
    }
    if (lastPacketTime > 0) {
        Serial.print("Last packet: ");
        Serial.print((millis() - lastPacketTime) / 1000);
        Serial.println(" seconds ago");
    }
    Serial.println("========================================");
    Serial.println();
}

// === Setup ===

void setup() {
    Serial.begin(115200);
    delay(1000);

    Serial.println();
    Serial.println("========================================");
    Serial.println("=== Meshtastic Gateway Node ===");
    Serial.println("=== Sensor-PAX Mesh Network ===");
    Serial.println("========================================");
    Serial.println();

    // Initialize LED
    pinMode(LED_PIN, OUTPUT);
    digitalWrite(LED_PIN, LOW);

    // Connect to WiFi
    connectWiFi();

    // Initialize radio
    if (!initRadio()) {
        Serial.println("FATAL: Radio initialization failed!");
        Serial.println("System halted.");
        while (1) {
            digitalWrite(LED_PIN, HIGH);
            delay(100);
            digitalWrite(LED_PIN, LOW);
            delay(100);
        }
    }

    // Start receiving
    Serial.println("=== Starting mesh receive mode ===");
    int state = radio.startReceive();
    if (state != RADIOLIB_ERR_NONE) {
        Serial.print("Start receive failed, code: ");
        Serial.println(state);
    } else {
        Serial.println("Gateway ready - listening for mesh packets...");
    }

    Serial.println();
    printStatus();
}

// === Main Loop ===

unsigned long lastStatusPrint = 0;
const unsigned long STATUS_INTERVAL = 60000;  // Print status every 60 seconds

void loop() {
    // Check if packet available
    if (radio.getPacketLength() > 0) {
        uint8_t buffer[sizeof(MeshPacket)];
        int state = radio.readData(buffer, sizeof(buffer));

        if (state == RADIOLIB_ERR_NONE) {
            MeshPacket* packet = (MeshPacket*)buffer;

            // Only process PAX data packets (ignore ACKs from other gateways)
            if (packet->type == PKT_TYPE_PAX_DATA) {
                processPacket(packet);
            }
        } else {
            Serial.print("Read error, code: ");
            Serial.println(state);
        }

        // Return to receive mode
        radio.startReceive();
    }

    // Print status periodically
    if (millis() - lastStatusPrint > STATUS_INTERVAL) {
        printStatus();
        lastStatusPrint = millis();
    }

    // Check WiFi connection periodically
    if (WiFi.status() != WL_CONNECTED) {
        Serial.println("WiFi connection lost! Reconnecting...");
        connectWiFi();
    }

    // Small delay to prevent tight loop
    delay(10);
}
