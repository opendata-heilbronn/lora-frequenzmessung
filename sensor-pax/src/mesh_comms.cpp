#include "mesh_comms.h"
#include "logging.h"
#include "customs.h"
#include <RadioLib.h>
#include <time.h>

// === Hardware Pin Definitions for Heltec WiFi LoRa 32 V3 ===
// These match the SX1262 radio configuration on the Heltec board
#define RADIO_SCLK_PIN 9
#define RADIO_MISO_PIN 11
#define RADIO_MOSI_PIN 10
#define RADIO_CS_PIN 8
#define RADIO_DIO1_PIN 14
#define RADIO_RST_PIN 12
#define RADIO_BUSY_PIN 13

// Global radio instance for mesh communications
// Using SX1262 module from RadioLib
static SX1262* meshRadio = nullptr;
static bool meshRadioInitialized = false;

// === Initialization ===

bool initMeshComms() {
    if (meshRadioInitialized) {
        logMessage("Mesh comms already initialized");
        return true;
    }

    logMessage("Initializing mesh communications...");

    // Create new SX1262 instance with Heltec V3 pins
    meshRadio = new SX1262(new Module(RADIO_CS_PIN, RADIO_DIO1_PIN, RADIO_RST_PIN, RADIO_BUSY_PIN));

    // Initialize radio with mesh parameters
    int state = meshRadio->begin(
        MESH_FREQUENCY,           // Frequency (MHz)
        MESH_BANDWIDTH,           // Bandwidth (kHz)
        MESH_SPREADING_FACTOR,    // Spreading factor
        MESH_CODING_RATE,         // Coding rate
        0x12,                     // Sync word (0x12 for private networks)
        MESH_TX_POWER,            // TX power (dBm)
        MESH_PREAMBLE_LENGTH,     // Preamble length
        0                         // TCXO voltage (0 = not used)
    );

    if (state != RADIOLIB_ERR_NONE) {
        logMessage("Mesh radio init failed, code: " + String(state));
        delete meshRadio;
        meshRadio = nullptr;
        return false;
    }

    // Configure for explicit header mode (needed for variable packet sizes)
    meshRadio->explicitHeader();

    // Set CRC enabled for data integrity
    meshRadio->setCRC(true);

    meshRadioInitialized = true;
    logMessage("Mesh radio initialized successfully");
    logMessage("Frequency: " + String(MESH_FREQUENCY, 3) + " MHz, SF: " + String(MESH_SPREADING_FACTOR));

    return true;
}

void deinitMeshComms() {
    if (!meshRadioInitialized) {
        return;
    }

    logMessage("Deinitializing mesh communications...");

    if (meshRadio != nullptr) {
        // Put radio to sleep to save power
        meshRadio->sleep();
        delete meshRadio;
        meshRadio = nullptr;
    }

    meshRadioInitialized = false;
    logMessage("Mesh radio deinitialized");
}

// === CRC Calculation ===

uint16_t calculateCRC16(const uint8_t* data, size_t length) {
    // CRC-16-CCITT algorithm (polynomial 0x1021)
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
    if (packet == nullptr) {
        return false;
    }

    // Calculate CRC over packet data (excluding the CRC field itself)
    size_t dataLength = sizeof(MeshPacket) - sizeof(packet->crc);
    uint16_t calculatedCRC = calculateCRC16((const uint8_t*)packet, dataLength);

    return (calculatedCRC == packet->crc);
}

// === Packet Creation ===

MeshPacket createMeshPacket(const char* sensor_id, uint16_t pax_count, uint8_t battery) {
    MeshPacket packet;

    // Set packet type
    packet.type = PKT_TYPE_PAX_DATA;

    // Copy sensor ID (ensure null termination)
    memset(packet.sensor_id, 0, sizeof(packet.sensor_id));
    strncpy(packet.sensor_id, sensor_id, sizeof(packet.sensor_id) - 1);

    // Set data fields
    packet.pax_count = pax_count;
    packet.battery = battery;
    packet.hop_count = 0;  // Direct transmission (no hops yet)

    // Set timestamp (seconds since epoch)
    // Note: ESP32 RTC is not synchronized, so this is just millis() / 1000
    // In a production system, you'd use NTP or GPS time
    packet.timestamp = millis() / 1000;

    // Calculate and set CRC (over all fields except CRC itself)
    size_t dataLength = sizeof(MeshPacket) - sizeof(packet.crc);
    packet.crc = calculateCRC16((const uint8_t*)&packet, dataLength);

    return packet;
}

// === Mesh Transmission ===

bool tryMeshTransmit(int pax_count, float battery) {
    // Initialize mesh radio if not already done
    if (!meshRadioInitialized) {
        if (!initMeshComms()) {
            logMessage("Failed to initialize mesh comms");
            return false;
        }
    }

    logMessage("=== Attempting mesh transmission ===");
    logMessage("PAX: " + String(pax_count) + ", Battery: " + String(battery, 1) + "%");

    // Create packet
    uint8_t batteryByte = (uint8_t)constrain(battery, 0, 100);
    MeshPacket packet = createMeshPacket(sensor_id, (uint16_t)pax_count, batteryByte);

    // Log packet details
    logMessage("Packet size: " + String(sizeof(MeshPacket)) + " bytes");
    logMessage("Timestamp: " + String(packet.timestamp));
    logMessage("CRC: 0x" + String(packet.crc, HEX));

    // Attempt transmission with retry logic
    for (int retry = 0; retry <= MESH_MAX_RETRIES; retry++) {
        if (retry > 0) {
            logMessage("Retry attempt " + String(retry) + "/" + String(MESH_MAX_RETRIES));
            delay(500);  // Small delay between retries
        }

        // Transmit packet
        logMessage("Transmitting mesh packet...");
        int state = meshRadio->transmit((uint8_t*)&packet, sizeof(MeshPacket));

        if (state != RADIOLIB_ERR_NONE) {
            logMessage("Transmission failed, code: " + String(state));
            continue;  // Try again if retries left
        }

        logMessage("Packet transmitted, waiting for ACK...");

        // Switch to RX mode to wait for ACK
        meshRadio->startReceive();

        // Wait for ACK with timeout
        unsigned long ackStartTime = millis();
        bool ackReceived = false;

        while (millis() - ackStartTime < MESH_ACK_TIMEOUT) {
            // Check if data available
            if (meshRadio->getPacketLength() > 0) {
                // Read received data
                uint8_t ackBuffer[sizeof(MeshPacket)];
                int ackState = meshRadio->readData(ackBuffer, sizeof(ackBuffer));

                if (ackState == RADIOLIB_ERR_NONE) {
                    // Check if it's an ACK packet
                    MeshPacket* ackPacket = (MeshPacket*)ackBuffer;

                    if (ackPacket->type == PKT_TYPE_ACK) {
                        // Verify it's ACK for our sensor
                        if (strcmp(ackPacket->sensor_id, sensor_id) == 0) {
                            logMessage("ACK received from gateway!");
                            ackReceived = true;
                            break;
                        } else {
                            logMessage("ACK for different sensor: " + String(ackPacket->sensor_id));
                        }
                    }
                }

                // Start receive again after reading
                meshRadio->startReceive();
            }

            delay(10);  // Small delay to avoid tight loop
        }

        if (ackReceived) {
            logMessage("=== Mesh transmission successful ===");
            meshRadio->standby();  // Return to standby mode
            return true;
        } else {
            logMessage("No ACK received within " + String(MESH_ACK_TIMEOUT) + "ms");
        }

        // Put radio in standby before next retry
        meshRadio->standby();
    }

    // All retries exhausted
    logMessage("=== Mesh transmission failed after " + String(MESH_MAX_RETRIES + 1) + " attempts ===");
    return false;
}
