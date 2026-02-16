# Meshtastic Gateway Node

WiFi-enabled gateway that receives PAX count data from mesh sensors and forwards to web service.

## Hardware Requirements

- **Board**: Heltec WiFi LoRa 32 V3 (ESP32-S3 + SX1262)
- **Power**: USB or 3.7V LiPo battery
- **Connectivity**: WiFi required for HTTP forwarding

## Quick Start

### 1. Configure Gateway

Edit `src/config.h` and update:

```cpp
// WiFi credentials
#define WIFI_SSID "YourNetworkSSID"
#define WIFI_PASSWORD "YourNetworkPassword"

// API endpoint
#define API_HOST "your-service.com"
#define API_PORT 443
#define API_PATH "/api/pax"
#define API_USE_HTTPS true

// Optional: Authentication token
#define API_TOKEN "Bearer YOUR_TOKEN_HERE"
```

### 2. Build and Upload

```bash
cd gateway-node
pio run -t upload
pio device monitor
```

### 3. Verify Operation

After upload, you should see:

```
=== Meshtastic Gateway Node ===
=== Connecting to WiFi ===
WiFi connected!
IP address: 192.168.1.100
=== Initializing LoRa radio ===
Radio initialized successfully
Gateway ready - listening for mesh packets...
```

## LED Indicators

| Pattern | Meaning |
|---------|---------|
| Fast blink (500ms) | Connecting to WiFi |
| 3 quick flashes | WiFi connected successfully |
| Single flash (50ms) | Mesh packet received |
| Continuous fast blink | Radio initialization failed |

## API Integration

### HTTP POST Format

Gateway sends JSON POST requests:

```json
{
  "sensor": "863f75b0",
  "pax": 42,
  "battery": 85,
  "source": "mesh",
  "timestamp": 1708000000,
  "hop_count": 0
}
```

### Expected Response

- **Success**: HTTP 200, 201, or 204
- **Failure**: Any other code triggers retry (max 3 attempts)

### Example Backend (Node.js/Express)

```javascript
app.post('/api/pax', (req, res) => {
  const { sensor, pax, battery, source, timestamp } = req.body;

  // Store in database
  db.insert({
    sensor_id: sensor,
    count: pax,
    battery_pct: battery,
    source: source,
    received_at: new Date(timestamp * 1000)
  });

  res.status(200).json({ status: 'ok' });
});
```

## Mesh Configuration

Gateway must use **identical mesh parameters** to sensors:

| Parameter | Value | Notes |
|-----------|-------|-------|
| Frequency | 869.525 MHz | EU868 license-free |
| Bandwidth | 125 kHz | Standard LoRa |
| Spreading Factor | 9 | Balance range/speed |
| Coding Rate | 4/7 | Forward error correction |
| Sync Word | 0x12 | Private network |
| TX Power | 17 dBm | Gateway (sensors use 14 dBm) |

**IMPORTANT**: If you change mesh parameters in `sensor-pax/src/mesh_comms.h`, you **must** update `gateway-node/src/config.h` to match!

## Deployment Strategies

### Strategy 1: Central Gateway
- Deploy 1 gateway at central location
- Covers sensors within ~1-5 km (depending on terrain)
- Best for dense deployments

### Strategy 2: Distributed Gateways
- Deploy multiple gateways across area
- Redundancy: multiple gateways can receive same packet
- Backend deduplicates based on `sensor_id` + `timestamp`

### Strategy 3: Mobile Gateway
- Run gateway on laptop/Raspberry Pi
- Walk through area to collect data from distant sensors
- Useful for temporary events

## Troubleshooting

### WiFi Won't Connect

1. Check SSID/password in `config.h`
2. Verify WiFi is 2.4 GHz (ESP32 doesn't support 5 GHz)
3. Check serial monitor for timeout messages
4. Try reducing `WIFI_CONNECT_TIMEOUT` if router slow

### No Packets Received

1. Verify sensor is transmitting (check sensor logs)
2. Check mesh frequency matches sensors (`869.525 MHz`)
3. Verify sensors within range (~1-5 km line-of-sight)
4. Check antenna connected properly
5. Try increasing sensor `MESH_TX_POWER` to 17 dBm

### HTTP POST Fails

1. Verify API endpoint in `config.h`
2. Test manually: `curl -X POST https://your-service.com/api/pax -d '{"sensor":"test","pax":10,"battery":100,"source":"mesh","timestamp":123,"hop_count":0}' -H "Content-Type: application/json"`
3. Check WiFi signal strength (RSSI in status)
4. Verify firewall allows outbound HTTPS (port 443)
5. Check API authentication token if required

### Packets Received But Not Forwarded

1. Check serial monitor for HTTP response codes
2. Verify backend returns 200/201/204
3. Check `packetsFailed` in status output
4. Verify WiFi stable (not reconnecting frequently)

## Status Monitoring

Gateway prints status every 60 seconds:

```
=== Gateway Status ===
WiFi: Connected (-45 dBm, IP: 192.168.1.100)
Uptime: 3600 seconds
Packets received: 24
Packets forwarded: 23
Packets failed: 1
Success rate: 95.8%
Last packet: 120 seconds ago
```

## Power Consumption

| State | Current | Notes |
|-------|---------|-------|
| WiFi + LoRa RX | ~80-120 mA | Continuous operation |
| WiFi + HTTP POST | ~150-200 mA | Brief spikes |
| Daily (24h) | ~2.4 Ah | Use 5V/2A power supply |

**Recommendation**: Power via USB (not battery) for reliable 24/7 operation.

## Advanced Configuration

### Custom Packet Buffer

If HTTP fails frequently, increase buffer size:

```cpp
// config.h
#define MAX_PACKET_BUFFER 200  // Default: 100
```

### HTTP Retry Settings

Adjust retry behavior:

```cpp
// config.h
#define HTTP_MAX_RETRIES 5      // Default: 3
#define HTTP_RETRY_DELAY 2000   // Default: 1000 (ms)
```

### Status Print Interval

Change how often status is printed:

```cpp
// main.cpp line 371
const unsigned long STATUS_INTERVAL = 300000;  // 5 minutes (default: 60000)
```

## Security Considerations

1. **WiFi Security**: Use WPA2/WPA3 encrypted networks
2. **API Authentication**: Use `API_TOKEN` for authenticated endpoints
3. **HTTPS**: Always use `API_USE_HTTPS true` in production
4. **Network Isolation**: Consider separate VLAN for IoT devices

## Upgrading Firmware

```bash
# Pull latest changes
git pull

# Rebuild and upload
cd gateway-node
pio run -t upload
```

**Note**: Configuration in `config.h` is preserved between uploads.

## Support

For issues or questions:
1. Check serial monitor output (`pio device monitor`)
2. Review this README and troubleshooting section
3. Open GitHub issue with logs and configuration

## License

Same as parent sensor-pax project.
