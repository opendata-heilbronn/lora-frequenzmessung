#ifndef MESH_COMMS_H
#define MESH_COMMS_H

#include <Arduino.h>

// === Mesh Configuration Parameters ===
// These can be tuned based on deployment requirements

// Frequency for mesh communication (MHz) - EU868 license-free band
// Using 869.525 MHz which is outside LoRaWAN channels to avoid conflicts
#define MESH_FREQUENCY 869.525f

// LoRa modulation parameters
#define MESH_BANDWIDTH 125.0f           // kHz - standard LoRa bandwidth
#define MESH_SPREADING_FACTOR 9         // SF9 - balance between range and speed
#define MESH_CODING_RATE 7              // 4/7 forward error correction
#define MESH_TX_POWER 14                // dBm (max for EU without licensing)
#define MESH_PREAMBLE_LENGTH 8          // Standard preamble

// Timing parameters
#define MESH_ACK_TIMEOUT 5000           // ms to wait for gateway ACK
#define MESH_MAX_RETRIES 1              // Retry once before LoRaWAN fallback
#define MESH_RX_TIMEOUT 500             // ms to wait for ACK reception

// === Mesh Packet Structure ===
// Optimized for minimal airtime while including necessary data
#pragma pack(push, 1)  // Ensure tight packing (no padding)

struct MeshPacket {
    uint8_t type;               // Packet type (0x01 = PAX_DATA, 0x02 = ACK)
    char sensor_id[16];         // Sensor identifier (e.g., "863f75b0")
    uint16_t pax_count;         // Raw PAX count from BLE scan
    uint8_t battery;            // Battery percentage (0-100)
    uint8_t hop_count;          // Hop counter for relay limiting (future use)
    uint32_t timestamp;         // Unix timestamp (seconds since epoch)
    uint16_t crc;               // CRC16 checksum for data integrity
};

#pragma pack(pop)

// Packet types
#define PKT_TYPE_PAX_DATA 0x01
#define PKT_TYPE_ACK 0x02

// === Public Functions ===

/**
 * Initialize mesh communication module
 * Configures the SX1262 radio for P2P mode
 *
 * @return true if initialization successful, false otherwise
 */
bool initMeshComms();

/**
 * Attempt to transmit PAX data via mesh network
 * Will wait for ACK from gateway with timeout
 *
 * @param pax_count Raw PAX count from BLE scan
 * @param battery Battery percentage (0-100)
 * @return true if transmission successful (ACK received), false otherwise
 */
bool tryMeshTransmit(int pax_count, float battery);

/**
 * Deinitialize mesh communications and restore radio to LoRaWAN mode
 */
void deinitMeshComms();

// === Internal Functions (for implementation) ===

/**
 * Create a mesh packet with provided data
 * Automatically fills timestamp and calculates CRC
 *
 * @param sensor_id Sensor identifier string
 * @param pax_count PAX count value
 * @param battery Battery percentage
 * @return Populated MeshPacket structure
 */
MeshPacket createMeshPacket(const char* sensor_id, uint16_t pax_count, uint8_t battery);

/**
 * Calculate CRC16 checksum for data integrity
 * Uses CRC-16-CCITT algorithm (polynomial 0x1021)
 *
 * @param data Pointer to data buffer
 * @param length Length of data in bytes
 * @return 16-bit CRC value
 */
uint16_t calculateCRC16(const uint8_t* data, size_t length);

/**
 * Validate received packet CRC
 *
 * @param packet Pointer to received packet
 * @return true if CRC valid, false otherwise
 */
bool validatePacketCRC(const MeshPacket* packet);

#endif // MESH_COMMS_H
