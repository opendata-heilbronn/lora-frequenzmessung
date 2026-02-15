#ifndef _MAC_COUNTER_H
#define _MAC_COUNTER_H

#include <stdint.h>

/**
 * MAC Counter Module
 *
 * Efficient bitmap-based counting of unique MAC addresses.
 * Uses last 2 bytes of MAC address as a 16-bit hash for deduplication.
 * Only counts locally administered (random) MAC addresses.
 */

/**
 * Initialize/reset the MAC counter
 * Clears the bitmap and resets all counts to zero
 */
void mac_counter_reset(void);

/**
 * Add a MAC address to the counter
 *
 * @param mac_addr Pointer to 6-byte MAC address
 * @param rssi Signal strength (for filtering)
 * @param rssi_threshold Minimum RSSI to count (-80 dBm typical)
 * @return 1 if MAC was new and counted, 0 if already seen or filtered
 */
int mac_counter_add(const uint8_t* mac_addr, int rssi, int rssi_threshold);

/**
 * Get the current count of unique BLE MACs
 * @return Number of unique BLE MAC addresses seen
 */
uint16_t mac_counter_get_count(void);

#endif // _MAC_COUNTER_H
