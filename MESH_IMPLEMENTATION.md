# Meshtastic Mesh Networking Implementation

## Overview

The sensor-pax project now supports **dual-mode communication**:

1. **Primary**: Mesh networking (LoRa P2P) for decentralized sensor-to-sensor communication
2. **Fallback**: LoRaWAN/TTN for areas without mesh gateway coverage

This hybrid approach enables:
- ✅ Gradual deployment without requiring immediate mesh infrastructure
- ✅ Maximum reliability (always has fallback communication path)
- ✅ Network effect: mesh coverage naturally improves as more nodes deploy
- ✅ Backwards compatibility with existing LoRaWAN infrastructure

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Sensor Node (Dual-Mode)                 │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ 1. BLE Scan (30s) → PAX Count                        │   │
│  │ 2. Try Mesh Transmission (5s timeout)                │   │
│  │    ├─ Success → ACK received → Done                  │   │
│  │    └─ Fail → Fallback to LoRaWAN                     │   │
│  │ 3. Deep Sleep (~870s)                                │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
          │                                  │
          │ Mesh (LoRa P2P)                 │ LoRaWAN OTAA
          │ 869.525 MHz, SF9                │ EU868
          ▼                                  ▼
┌──────────────────────┐            ┌──────────────────┐
│  Gateway Node        │            │  TTN Gateway     │
│  (WiFi + LoRa)       │            │  (LoRaWAN)       │
│  - Receives mesh     │            │                  │
│  - Sends ACK         │            │                  │
│  - Forwards to API   │            │                  │
└──────────────────────┘            └──────────────────┘
          │                                  │
          │ HTTP POST                        │ Webhook
          ▼                                  ▼
┌─────────────────────────────────────────────────────┐
│           Web Service / Backend                     │
│  - Receives data from both mesh and LoRaWAN        │
│  - Tracks source: "mesh" vs "lorawan"              │
│  - Deduplicates if needed                          │
└─────────────────────────────────────────────────────┘
```

## Implementation Details

### Sensor Firmware Changes

**Files Modified:**
- `sensor-pax/platformio.ini` - Added RadioLib dependency
- `sensor-pax/src/main.cpp` - Added mesh-first logic with fallback

**Files Created:**
- `sensor-pax/src/mesh_comms.h` - Mesh packet definitions and API
- `sensor-pax/src/mesh_comms.cpp` - Mesh communication implementation

**Communication Flow:**

```cpp
void loop() {
    int pax_count = ScanPAX();        // BLE scan
    DeinitPAX();                       // Release BLE resources

    if (pax_count >= MIN_PAX_TO_SEND) {
        // Try mesh first (5 second timeout)
        bool meshSuccess = tryMeshTransmit(pax_count, battery);

        if (meshSuccess) {
            // Mesh worked - done!
            logMessage("Data sent via mesh");
        } else {
            // Mesh failed - fall back to LoRaWAN
            logMessage("Falling back to LoRaWAN");
            LoopLORA(pax_count, battery);
        }
    }

    enterDeepSleep();  // Sleep for ~870s
}
```

### Gateway Node

**Location:** `gateway-node/`

**Purpose:**
- Receives mesh packets from sensors
- Sends ACK back to sensors (so they don't fall back to LoRaWAN)
- Forwards data to web service via HTTP POST

**Key Features:**
- WiFi connectivity required
- Buffers packets if HTTP fails (retry logic)
- Status monitoring (packets received/forwarded/failed)
- LED indicators for status

**See:** `gateway-node/README.md` for detailed setup instructions

## Mesh Protocol Specification

### Packet Structure

```cpp
struct MeshPacket {
    uint8_t type;          // 0x01 = PAX_DATA, 0x02 = ACK
    char sensor_id[16];    // Sensor identifier (e.g., "863f75b0")
    uint16_t pax_count;    // Raw PAX count from BLE scan
    uint8_t battery;       // Battery percentage (0-100)
    uint8_t hop_count;     // Hop counter (future use for relay)
    uint32_t timestamp;    // Unix timestamp
    uint16_t crc;          // CRC16 checksum
};  // Total: 29 bytes
```

### Mesh Parameters

| Parameter | Value | Notes |
|-----------|-------|-------|
| Frequency | 869.525 MHz | EU868 license-free, outside LoRaWAN channels |
| Bandwidth | 125 kHz | Standard LoRa |
| Spreading Factor | SF9 | Balance: range ~1-5 km, airtime ~370ms |
| Coding Rate | 4/7 | Forward error correction |
| TX Power | 14 dBm (sensor), 17 dBm (gateway) | EU max without license |
| Sync Word | 0x12 | Private network (not public LoRaWAN) |
| Preamble | 8 symbols | Standard |

### Transmission Sequence

```
Sensor                          Gateway
  │                                │
  ├─── PAX_DATA packet ───────────>│
  │    (29 bytes, ~370ms airtime) │
  │                                ├─ Validate CRC
  │                                ├─ Process packet
  │<──────── ACK packet ───────────┤
  │    (29 bytes, ~370ms airtime) │
  │                                ├─ Forward to HTTP API
  │                                │
  └─ Success! (total ~1s)          └─ Return to RX mode
```

If no ACK received within 5 seconds → retry once → fallback to LoRaWAN

## Power Consumption Analysis

### Current Implementation (Mesh-First)

| State | Current | Duration | mAh/cycle |
|-------|---------|----------|-----------|
| Deep sleep | 7 µA | 870s | 0.00169 |
| BLE scan | 25 mA (avg) | 30s | 0.208 |
| Mesh TX | 100 mA | 1s | 0.028 |
| Mesh RX (ACK wait) | 15 mA | 1s | 0.004 |
| **Total (mesh success)** | | 902s | **0.242 mAh** |

**Mesh Failure Scenario** (falls back to LoRaWAN):

| Additional State | Current | Duration | mAh |
|------------------|---------|----------|-----|
| Mesh timeout | 15 mA | 5s | 0.021 |
| LoRaWAN TX | 120 mA | 2s | 0.067 |
| **Total (fallback)** | | | **0.330 mAh** |

**Daily Consumption:**
- **Mesh-only**: 0.242 mAh × 96 cycles = **23.2 mAh/day** ✅
- **Fallback mode**: 0.330 mAh × 96 cycles = **31.7 mAh/day** (still acceptable)

**Battery Life** (3000 mAh LiPo):
- Mesh-only: **~130 days**
- Fallback: **~95 days**
- Current LoRaWAN-only: **~113 days**

**Verdict**: Mesh mode is actually more efficient than pure LoRaWAN!

## Deployment Guide

### Phase 1: Deploy Sensors (LoRaWAN Fallback Mode)

Deploy sensors first **without** gateways:

```bash
cd sensor-pax
pio run -t upload
```

**Result**: All sensors use LoRaWAN fallback → data flows via TTN (existing infrastructure)

### Phase 2: Deploy First Gateway

Configure and deploy gateway:

```bash
cd gateway-node
# Edit src/config.h (WiFi, API endpoint)
pio run -t upload
```

**Result**: Sensors within range (~1-5 km) automatically switch to mesh

### Phase 3: Monitor Mesh Adoption

Check backend logs for `source` field:

```json
// Mesh traffic
{"sensor": "863f75b0", "pax": 42, "source": "mesh"}

// LoRaWAN fallback traffic
{"sensor": "863f75b0", "pax": 42, "source": "lorawan"}
```

**Metrics to track:**
- % of traffic via mesh vs LoRaWAN
- Sensors never using mesh (may be out of range)
- Gateway success rate

### Phase 4: Expand Gateway Coverage

Deploy additional gateways to improve mesh coverage:

- Place gateways at high points (rooftops, towers)
- Cover areas with many sensors
- Gateway range: ~1-5 km (line-of-sight), ~500m-2km (urban)

### Phase 5: Optimize

**If mesh adoption low:**
- Increase sensor `MESH_TX_POWER` to 17 dBm (max EU)
- Increase gateway `MESH_TX_POWER` (already at 17 dBm)
- Add more gateways in coverage gaps

**If mesh adoption high:**
- Consider disabling LoRaWAN fallback for well-covered areas (saves power)
- Deploy relay nodes (future feature) to extend range

## Backend Integration

### Handling Dual-Source Data

Your backend receives data from two sources:

```javascript
// Express.js example
app.post('/api/pax', (req, res) => {
  const { sensor, pax, battery, source, timestamp } = req.body;

  // Store with source tracking
  db.insert({
    sensor_id: sensor,
    count: pax,
    battery_pct: battery,
    source: source,  // "mesh" or "lorawan"
    timestamp: new Date(timestamp * 1000),
    received_at: new Date()
  });

  // Track mesh adoption metrics
  metrics.increment('pax_data_received', { source });

  res.status(200).json({ status: 'ok' });
});
```

### Deduplication (If Multiple Gateways)

If multiple gateways receive same packet:

```javascript
// Check for duplicate (sensor + timestamp)
const existing = await db.findOne({
  sensor_id: sensor,
  timestamp: new Date(timestamp * 1000)
});

if (existing) {
  console.log('Duplicate packet, ignoring');
  return res.status(200).json({ status: 'duplicate' });
}

// Insert new record
await db.insert({ ... });
```

### Monitoring Mesh Health

Query to check sensors not using mesh:

```sql
-- Sensors that haven't used mesh in last 24 hours
SELECT sensor_id, COUNT(*) as fallback_count
FROM pax_data
WHERE timestamp > NOW() - INTERVAL '24 hours'
  AND source = 'lorawan'
GROUP BY sensor_id
HAVING COUNT(*) > 10
ORDER BY fallback_count DESC;
```

## Configuration Reference

### Sensor Configuration

**File:** `sensor-pax/src/mesh_comms.h`

```cpp
// Tunable parameters
#define MESH_FREQUENCY 869.525f      // MHz
#define MESH_BANDWIDTH 125.0f        // kHz
#define MESH_SPREADING_FACTOR 9      // 7-12 (higher = longer range, slower)
#define MESH_TX_POWER 14             // dBm (max 17 for EU)
#define MESH_ACK_TIMEOUT 5000        // ms to wait for ACK
#define MESH_MAX_RETRIES 1           // Retry before fallback
```

**Range vs Speed Tradeoff:**

| SF | Range | Airtime | Battery Impact |
|----|-------|---------|----------------|
| 7  | Short (~500m) | 41ms | Best |
| 9  | Medium (~2km) | 370ms | Good ✅ |
| 11 | Long (~5km) | 1.5s | Acceptable |
| 12 | Max (~10km) | 2.8s | High |

**Recommendation**: Keep SF9 unless you need extreme range

### Gateway Configuration

**File:** `gateway-node/src/config.h`

Must match sensor configuration exactly!

## Troubleshooting

### Sensors Always Fall Back to LoRaWAN

**Symptoms:** Backend shows 100% `source: "lorawan"`, 0% `source: "mesh"`

**Possible Causes:**
1. Gateway not running
2. Gateway out of range (>5km)
3. Mesh frequency mismatch
4. Poor antenna on sensor or gateway
5. Physical obstacles (buildings, hills)

**Solutions:**
1. Check gateway serial monitor - should show "listening for mesh packets"
2. Deploy gateway closer to sensors
3. Verify `MESH_FREQUENCY` matches in both sensor and gateway
4. Check antenna connections
5. Increase `MESH_TX_POWER` to 17 dBm

### Gateway Receives Packets But Doesn't Send ACK

**Symptoms:** Gateway logs "packet received" but sensor still falls back

**Possible Causes:**
1. Gateway fails to transmit ACK
2. Sensor doesn't receive ACK (weak signal)
3. ACK timing issue

**Solutions:**
1. Check gateway logs for "ACK sent successfully"
2. Increase gateway antenna height
3. Increase `MESH_ACK_TIMEOUT` in sensor

### Mesh Works But HTTP Forwarding Fails

**Symptoms:** Gateway receives packets, sends ACK, but `packetsFailed` increases

**Possible Causes:**
1. WiFi disconnected
2. API endpoint wrong
3. API returns error code
4. Firewall blocks outbound HTTPS

**Solutions:**
1. Check gateway WiFi status in logs
2. Verify `API_HOST`, `API_PORT`, `API_PATH` in `config.h`
3. Check API server logs for errors
4. Test manually with curl (see gateway README)

### High Battery Drain on Sensors

**Symptoms:** Battery lasts <50 days (vs expected 95-130 days)

**Possible Causes:**
1. Sensors repeatedly timeout waiting for mesh ACK
2. Falling back to LoRaWAN too often
3. Deep sleep not working

**Solutions:**
1. Check % of mesh vs LoRaWAN in backend
2. Reduce `MESH_ACK_TIMEOUT` to 3000ms (saves power on timeout)
3. Monitor deep sleep current with multimeter (should be ~7µA)

## Future Enhancements

### 1. Multi-Hop Relay (Planned)

Enable sensors to relay packets from distant sensors:

```
Sensor A ──> Sensor B (relay) ──> Gateway
(5km away)   (2.5km from both)
```

**Implementation:**
- Check `hop_count` field in packet
- If `hop_count < MAX_HOPS`, sensor can relay
- Prevents loops with message ID deduplication

### 2. Mesh Time Synchronization (Planned)

Gateway broadcasts time sync packets:
- Sensors update RTC from gateway
- Enables accurate timestamps without GPS/NTP

### 3. Mesh Configuration Updates (Planned)

Gateway sends configuration updates:
- Update scan interval
- Update privacy threshold
- Update factor value

Eliminates need to physically access sensors for config changes.

### 4. Battery Monitoring Dashboard (Planned)

Gateway reports sensor battery levels:
- Alert when battery < 20%
- Predict battery replacement date
- Identify sensors with unusual drain

## Testing Checklist

Before deployment:

- [ ] Sensor builds without errors
- [ ] Gateway builds without errors
- [ ] Gateway connects to WiFi
- [ ] Gateway forwards test packet to API
- [ ] Sensor transmits mesh packet (check gateway logs)
- [ ] Gateway sends ACK (check sensor logs "ACK received")
- [ ] Sensor falls back to LoRaWAN when gateway off
- [ ] Backend receives data with `source: "mesh"`
- [ ] Backend receives data with `source: "lorawan"`
- [ ] Deep sleep current measured (~7µA)
- [ ] Battery voltage reading accurate

## Support & Contributing

**Issues:** Open GitHub issue with:
- Sensor logs (`pio device monitor`)
- Gateway logs
- Configuration files
- Backend logs (if applicable)

**Contributing:**
- Test different spreading factors and report range results
- Submit power consumption measurements
- Report real-world deployment experiences

## References

- **Meshtastic Protocol**: https://meshtastic.org/docs/overview/mesh-algo/
- **RadioLib Documentation**: https://github.com/jgromes/RadioLib
- **LoRa Airtime Calculator**: https://www.thethingsnetwork.org/airtime-calculator
- **EU863-870 Frequency Plan**: https://www.thethingsnetwork.org/docs/lorawan/frequencies-by-country/

## License

Same as parent sensor-pax project.
