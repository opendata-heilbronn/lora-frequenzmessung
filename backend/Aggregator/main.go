package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-resty/resty/v2"
	"github.com/gofiber/fiber/v3"
	Misc2 "github.com/opendata-heilbronn/l
	Misc2 "github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Mqtt"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Yaml"
	structs2 "github.com/opendata-heilbronn/lora-frequenzmessung/structs"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

var broker string
var brokerUsername string
var brokerPassword string
var clientID string
var topic string

// MeshData represents the simplified mesh gateway payload
type MeshData struct {
	Sensor    string  `json:"sensor"`
	Pax       float64 `json:"pax"`
	Battery   float64 `json:"battery"`
	Source    string  `json:"source"`    // "mesh" or "lorawan"
	Timestamp int64   `json:"timestamp"` // Unix timestamp
	HopCount  int     `json:"hop_count"` // Mesh hop count
}

// processSensorData handles sensor data from any source (MQTT/HTTP) and forwards to backend
// This is the common processing logic shared by both TTN and mesh paths
func processSensorData(sensorID string, typeID int, value float64, source string) error {
	clients := Yaml.LoadYaml()
	resty := resty.New()

	// Find client by sensor ID
	clientOfMessage := structs2.Clients{}
	found := false
	for _, client := range clients {
		if client.UUID == sensorID {
			clientOfMessage = client
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("unknown sensor ID: %s", sensorID)
	}

	// Build data structure for backend
	densityData := structs2.DensityData{
		SensorID: sensorID,
		Value:    value,
	}

	var DataWithClient structs2.DensityDataWithClient
	switch typeID {
	case 0:
		DataWithClient.DataType = "densityData"
	case 1:
		DataWithClient.DataType = "batteryData"
	default:
		return fmt.Errorf("unknown type ID: %d", typeID)
	}

	DataWithClient.Client = clientOfMessage
	DataWithClient.Data = densityData

	// Marshal to JSON
	encodedData, err := json.Marshal(DataWithClient)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	fmt.Printf("[%s] Forwarding to backend: %s\n", source, string(encodedData))

	// Send to backend
	resp, err := resty.R().
		SetBody(encodedData).
		Post(fmt.Sprintf("%s/add-sensor-data", Misc2.GetBackendURL()))

	if err != nil {
		return fmt.Errorf("failed to send to backend: %v", err)
	}

	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("backend returned status %d: %s", resp.StatusCode(), resp.String())
	}

	fmt.Printf("[%s] Successfully forwarded %s data (value: %.2f) to backend\n",
		source, DataWithClient.DataType, value)
	return nil
}

// MQTT message handler for TTN messages (existing functionality)
var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Println("Received MQTT message: ", msg.MessageID())
	var ttnMessage structs2.TtnMessage

	err := json.Unmarshal(msg.Payload(), &ttnMessage)
	if err != nil {
		log.Printf("Unable to unmarshal TTN JSON: %s", err)
		return
	}

	// Decode base64 payload
	data, err := base64.StdEncoding.DecodeString(ttnMessage.UplinkMessage.FrmPayload)
	if err != nil {
		log.Printf("Failed to decode base64 from TTN Message: %v", err)
		return
	}

	// Remove last byte if it's null
	if len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}

	// Parse payload: "863f75b0,0,42.0000,1,85.0000"
	// Format: sensor_id,type,value,type,value,...
	stringSlice := strings.Split(string(data), ",")
	if len(stringSlice) < 3 {
		log.Printf("Invalid payload format: %s", string(data))
		return
	}

	sensorID := stringSlice[0]
	numberOfValues := (len(stringSlice) - 1) / 2

	fmt.Printf("[LoRaWAN] Processing %d values from sensor %s\n", numberOfValues, sensorID)

	// Process each type/value pair
	shift := 1
	for i := 0; i < numberOfValues; i++ {
		typeID, err := strconv.Atoi(stringSlice[shift])
		if err != nil {
			log.Printf("Invalid type ID: %s", stringSlice[shift])
			shift += 2
			continue
		}

		value, err := strconv.ParseFloat(stringSlice[shift+1], 64)
		if err != nil {
			log.Printf("Invalid value: %s", stringSlice[shift+1])
			shift += 2
			continue
		}

		// Process using common logic
		err = processSensorData(sensorID, typeID, value, "lorawan")
		if err != nil {
			log.Printf("Failed to process sensor data: %v", err)
		}

		shift += 2
	}
}

// HTTP handler for mesh gateway data (new functionality)
func handleMeshData(c fiber.Ctx) error {
	var meshData MeshData

	// Parse incoming JSON
	if err := c.Bind().JSON(&meshData); err != nil {
		log.Printf("Failed to parse mesh data: %v", err)
		return c.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf("Invalid JSON: %v", err))
	}

	fmt.Printf("[Mesh] Received data from sensor %s (pax: %.0f, battery: %.0f%%)\n",
		meshData.Sensor, meshData.Pax, meshData.Battery)

	// Process PAX data (type 0)
	if err := processSensorData(meshData.Sensor, 0, meshData.Pax, meshData.Source); err != nil {
		log.Printf("Failed to process PAX data: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to process PAX: %v", err))
	}

	// Process Battery data (type 1)
	if err := processSensorData(meshData.Sensor, 1, meshData.Battery, meshData.Source); err != nil {
		log.Printf("Failed to process battery data: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("Failed to process battery: %v", err))
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"status":  "accepted",
		"sensor":  meshData.Sensor,
		"source":  meshData.Source,
		"message": "Mesh data processed successfully",
	})
}

func main() {
	Misc2.StartUp()
	println("STARTING AGGREGATOR")
	println("=====================")
	println("Listening for:")
	println("  - TTN messages via MQTT (LoRaWAN)")
	println("  - Mesh gateway data via HTTP :3002/mesh-data")
	println("=====================")

	// Start HTTP server for mesh data
	app := fiber.New(fiber.Config{
		AppName: "Sensor-PAX Aggregator",
	})

	// Mesh data endpoint
	app.Post("/mesh-data", handleMeshData)

	// Health check endpoint
	app.Get("/health", func(c fiber.Ctx) error {
			"status":  "healthy",
			"status": "healthy",
			"service": "aggregator",
		})
	})

	// Start HTTP server in background
	go func() {
		err := app.Listen(":3002")
		if err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Start MQTT connection for TTN messages
	broker, clientID, topic, username, password, _ := Misc2.SetupVars()
	opts := mqtt.NewClientOptions()
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetUsername(username)
	opts.SetPassword(password)
	mqttClient := Mqtt.StartMqtttConnection(broker, clientID, opts)
	sub(mqttClient, topic)

	// Wait for shutdown signal
	signals()
}

func sub(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to MQTT topic: %s\n", topic)
}

func signals() {
	fmt.Println("Awaiting shutdown signal (Ctrl+C)...")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool, 1)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Printf("Received signal: %v\n", sig)
		fmt.Println("Shutting down gracefully...")
		done <- true
	}()

	<-done
	fmt.Println("Aggregator stopped")
}
