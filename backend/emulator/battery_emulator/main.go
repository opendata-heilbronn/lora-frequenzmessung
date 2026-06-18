package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"codeberg.org/cfhn/lorax.git/backend/pkg/misc"
	"codeberg.org/cfhn/lorax.git/backend/pkg/mqtt"
	"codeberg.org/cfhn/lorax.git/backend/pkg/structs"
	emqtt "github.com/eclipse/paho.mqtt.golang"
)

// LoRaWAN message structures.
type ApplicationIDs struct {
	ApplicationID string `json:"application_id"`
}

type EndDeviceIDs struct {
	DeviceID       string         `json:"device_id"`
	ApplicationIDs ApplicationIDs `json:"application_ids"`
	DevEUI         string         `json:"dev_eui"`
	DevAddr        string         `json:"dev_addr"`
}

type GatewayIDs struct {
	GatewayID string `json:"gateway_id"`
	EUI       string `json:"eui"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  int     `json:"altitude"`
	Source    string  `json:"source"`
}

type RxMetadata struct {
	GatewayIDs  GatewayIDs `json:"gateway_ids"`
	Time        string     `json:"time"`
	Timestamp   int64      `json:"timestamp"`
	RSSI        int        `json:"rssi"`
	ChannelRSSI int        `json:"channel_rssi"`
	SNR         float64    `json:"snr"`
	Location    Location   `json:"location"`
	ReceivedAt  string     `json:"received_at"`
}

type LoRaSettings struct {
	Bandwidth       int    `json:"bandwidth"`
	SpreadingFactor int    `json:"spreading_factor"`
	CodingRate      string `json:"coding_rate"`
}

type DataRate struct {
	LoRa LoRaSettings `json:"lora"`
}

type Settings struct {
	DataRate  DataRate `json:"data_rate"`
	Frequency string   `json:"frequency"`
	Timestamp int64    `json:"timestamp"`
	Time      string   `json:"time"`
}

type VersionIDs struct {
	BrandID         string `json:"brand_id"`
	ModelID         string `json:"model_id"`
	HardwareVersion string `json:"hardware_version"`
	FirmwareVersion string `json:"firmware_version"`
	BandID          string `json:"band_id"`
}

type NetworkIDs struct {
	NetID          string `json:"net_id"`
	NSID           string `json:"ns_id"`
	TenantID       string `json:"tenant_id"`
	ClusterID      string `json:"cluster_id"`
	ClusterAddress string `json:"cluster_address"`
}

type UplinkMessage struct {
	FPort           int          `json:"f_port"`
	FrmPayload      string       `json:"frm_payload"`
	MessageString   string       `json:"messageString"`
	RxMetadata      []RxMetadata `json:"rx_metadata"`
	Settings        Settings     `json:"settings"`
	ReceivedAt      string       `json:"received_at"`
	Confirmed       bool         `json:"confirmed"`
	ConsumedAirtime string       `json:"consumed_airtime"`
	VersionIDs      VersionIDs   `json:"version_ids"`
	NetworkIDs      NetworkIDs   `json:"network_ids"`
}

type LoRaWANMessage struct {
	EndDeviceIDs  EndDeviceIDs  `json:"end_device_ids"`
	ReceivedAt    string        `json:"received_at"`
	UplinkMessage UplinkMessage `json:"uplink_message"`
}

var (
	broker   string
	clientID string
	topic    string
)

var connectHandler emqtt.OnConnectHandler = func(client emqtt.Client) {
	fmt.Println("Connected to MQTT Broker")
}

var connectLostHandler emqtt.ConnectionLostHandler = func(client emqtt.Client, err error) {
	fmt.Printf("Connection lost: %v", err)
}

func main() {
	misc.StartUp()

	broker, clientID, topic, _, _, _ := misc.SetupVars()
	opts := emqtt.NewClientOptions()
	client := mqtt.StartMqtttConnection(broker, clientID, opts)

	for {
		uuid := "f4b894f9b0db"
		message := structs.BatteryChargeData{
			SensorID: uuid,
			Value:    float64(rand.Intn(10000)) / 100.0, // Generate values like 99.9999
		}
		messageString := base64.StdEncoding.EncodeToString([]byte(
			fmt.Sprintf("%s,%f,%dx", message.SensorID, message.Value, 1),
		))
		now := time.Now().UTC().Format(time.RFC3339Nano)
		timestamp := time.Now().Unix()

		lorawanMessage := LoRaWANMessage{
			EndDeviceIDs: EndDeviceIDs{
				DeviceID: "lora-dev",
				ApplicationIDs: ApplicationIDs{
					ApplicationID: "your-ttn-app-id",
				},
				DevEUI:  "0000000000000000",
				DevAddr: "00000000",
			},
			ReceivedAt: now,
			UplinkMessage: UplinkMessage{
				FPort:         2,
				FrmPayload:    messageString,
				MessageString: messageString,
				RxMetadata: []RxMetadata{
					{
						GatewayIDs: GatewayIDs{
							GatewayID: "example-gateway-001",
							EUI:       "0000000000000000",
						},
						Time:        now,
						Timestamp:   timestamp,
						RSSI:        -64,
						ChannelRSSI: -64,
						SNR:         9.25,
						Location: Location{
							Latitude:  0.0,
							Longitude: 0.0,
							Altitude:  57,
							Source:    "SOURCE_REGISTRY",
						},
						ReceivedAt: now,
					},
					{
						GatewayIDs: GatewayIDs{
							GatewayID: "example-gateway-002",
							EUI:       "0000000000000001",
						},
						Time:        now,
						Timestamp:   timestamp + 1000,
						RSSI:        -119,
						ChannelRSSI: -119,
						SNR:         -2.25,
						Location: Location{
							Latitude:  0.0,
							Longitude: 0.0,
							Altitude:  187,
							Source:    "SOURCE_REGISTRY",
						},
						ReceivedAt: now,
					},
				},
				Settings: Settings{
					DataRate: DataRate{
						LoRa: LoRaSettings{
							Bandwidth:       125000,
							SpreadingFactor: 7,
							CodingRate:      "4/5",
						},
					},
					Frequency: "867700000",
					Timestamp: timestamp,
					Time:      now,
				},
				ReceivedAt:      now,
				Confirmed:       true,
				ConsumedAirtime: "0.066816s",
				VersionIDs: VersionIDs{
					BrandID:         "heltec",
					ModelID:         "wifi-lora-32-class-a-otaa",
					HardwareVersion: "_unknown_hw_version_",
					FirmwareVersion: "1.0",
					BandID:          "EU_863_870",
				},
				NetworkIDs: NetworkIDs{
					NetID:          "000013",
					NSID:           "EC656E0000000181",
					TenantID:       "ttn",
					ClusterID:      "eu1",
					ClusterAddress: "eu1.cloud.thethings.network",
				},
			},
		}

		// Convert to JSON
		jsonMessage, err := json.Marshal(lorawanMessage)
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			continue
		}

		token := client.Publish(topic, 0, true, string(jsonMessage))
		token.Wait()
		fmt.Printf("Published LoRaWAN message: %s\n", string(jsonMessage))
		time.Sleep(time.Second)
	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
