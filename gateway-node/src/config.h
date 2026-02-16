#ifndef CONFIG_H
#define CONFIG_H

// === WiFi Configuration ===
// Update these with your actual WiFi credentials
#define WIFI_SSID "YourNetworkSSID"
#define WIFI_PASSWORD "YourNetworkPassword"

// WiFi connection timeout (milliseconds)
#define WIFI_CONNECT_TIMEOUT 30000

// === API Configuration ===
// HTTP endpoint for forwarding mesh data
// Example: "https://your-service.com" or "http://192.168.1.100:3000"
#define API_HOST "your-service.com"
#define API_PORT 443  // 443 for HTTPS, 80 for HTTP
#define API_PATH "/api/pax"
#define API_USE_HTTPS true  // true for HTTPS, false for HTTP

// Optional: API authentication token
// Leave empty ("") if no authentication required
#define API_TOKEN ""

// === Mesh Configuration ===
// Must match sensor configuration exactly
#define MESH_FREQUENCY 869.525f         // MHz
#define MESH_BANDWIDTH 125.0f           // kHz
#define MESH_SPREADING_FACTOR 9         // SF9
#define MESH_CODING_RATE 7              // 4/7
#define MESH_SYNC_WORD 0x12             // Private network sync word
#define MESH_PREAMBLE_LENGTH 8          // Preamble length

// === Hardware Pin Definitions ===
// Heltec WiFi LoRa 32 V3 SX1262 pins
#define RADIO_SCLK_PIN 9
#define RADIO_MISO_PIN 11
#define RADIO_MOSI_PIN 10
#define RADIO_CS_PIN 8
#define RADIO_DIO1_PIN 14
#define RADIO_RST_PIN 12
#define RADIO_BUSY_PIN 13

// LED pin for status indication (built-in LED)
#define LED_PIN 35

// === Gateway Behavior ===
// Maximum number of packets to buffer if HTTP fails
#define MAX_PACKET_BUFFER 100

// Retry settings for HTTP POST
#define HTTP_MAX_RETRIES 3
#define HTTP_RETRY_DELAY 1000  // ms between retries

// Status LED blink patterns
#define LED_BLINK_WIFI_CONNECTING 500   // ms - fast blink while connecting
#define LED_BLINK_PACKET_RECEIVED 100   // ms - quick flash on packet
#define LED_ON_DURATION 50               // ms - LED on time for flash

#endif // CONFIG_H
