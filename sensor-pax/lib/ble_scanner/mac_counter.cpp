/*
 * MAC Counter Implementation
 *
 * Based on libpax bitmap algorithm by Deutsche Bahn Station&Service AG
 * Simplified for BLE-only scanning with deep sleep support
 */

#include "mac_counter.h"
#include <string.h>
#include <Arduino.h>

typedef uint32_t bitmap_t;
static const int BITS_PER_WORD = sizeof(bitmap_t) * 8;

#define WORD_OFFSET(b) ((b) / BITS_PER_WORD)
#define BIT_OFFSET(b) ((b) % BITS_PER_WORD)
#define LIBPAX_MAX_SIZE 0xFFFF  // full enumeration of uint16_t
#define LIBPAX_MAP_SIZE (LIBPAX_MAX_SIZE / BITS_PER_WORD)

// Bitmap for tracking seen MAC address hashes
static bitmap_t seen_ids_map[LIBPAX_MAP_SIZE];
static uint16_t seen_count = 0;

static inline void set_id(bitmap_t* bitmap, uint16_t id) {
    bitmap[WORD_OFFSET(id)] |= ((bitmap_t)1 << BIT_OFFSET(id));
}

static inline int get_id(bitmap_t* bitmap, uint16_t id) {
    bitmap_t bit = bitmap[WORD_OFFSET(id)] & ((bitmap_t)1 << BIT_OFFSET(id));
    return bit != 0;
}

static int add_to_bucket(uint16_t id) {
    if (get_id(seen_ids_map, id)) {
        return 0;  // already seen
    } else {
        set_id(seen_ids_map, id);
        seen_count++;
        return 1;  // new
    }
}

void mac_counter_reset(void) {
    memset(seen_ids_map, 0, sizeof(seen_ids_map));
    seen_count = 0;
}

int mac_counter_add(const uint8_t* mac_addr, int rssi, int rssi_threshold) {
    // Filter by RSSI threshold
    if (rssi < rssi_threshold) {
        return 0;
    }

    // Only count locally administered (random) MAC addresses
    // Bit 1 of first byte indicates locally administered
    if (!(mac_addr[0] & 0b10)) {
        return 0;
    }

    // Use last 2 bytes of MAC as hash (same as libpax)
    uint16_t id = (mac_addr[4] << 8) | mac_addr[5];

    return add_to_bucket(id);
}

uint16_t mac_counter_get_count(void) {
    return seen_count;
}
