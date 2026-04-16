package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-resty/resty/v2"
	Misc2 "github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Mqtt"
	structs2 "github.com/opendata-heilbronn/lora-frequenzmessung/structs"
)

// sensorCache caches the sensor list from the backend API to avoid an HTTP
// round-trip on every MQTT message.
var sensorCache struct {
	sync.Mutex
	sensors   []structs2.Clients
	fetchedAt time.Time
}

const sensorCacheTTL = 30 * time.Second

const internalKeyHeader = "X-Internal-Key"

func loadSensors() ([]structs2.Clients, error) {
	sensorCache.Lock()
	defer sensorCache.Unlock()

	if time.Since(sensorCache.fetchedAt) < sensorCacheTTL && sensorCache.sensors != nil {
		return sensorCache.sensors, nil
	}

	backendURL := Misc2.GetBackendURL()
	resp, err := resty.New().R().
		SetHeader("Accept", "application/json").
		SetHeader(internalKeyHeader, Misc2.GetInternalAPIKey()).
		Get(fmt.Sprintf("%s/internal/sensors", backendURL))
	if err != nil {
		return nil, fmt.Errorf("fetch sensors from backend: %w", err)
	}

	var sensors []structs2.Clients
	if err := json.Unmarshal(resp.Body(), &sensors); err != nil {
		return nil, fmt.Errorf("unmarshal sensors: %w", err)
	}

	sensorCache.sensors = sensors
	sensorCache.fetchedAt = time.Now()
	return sensors, nil
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var ttnMessage structs2.TtnMessage
	restyClient := resty.New()

	clients, err := loadSensors()
	if err != nil {
		log.Printf("ERROR: failed to load sensors: %v", err)
		return
	}

	err = json.Unmarshal(msg.Payload(), &ttnMessage)
	if err != nil {
		log.Printf("ERROR: unable to unmarshal TTN JSON: %v", err)
		return
	}
	data, err := base64.StdEncoding.DecodeString(ttnMessage.UplinkMessage.FrmPayload)
	if err != nil {
		log.Printf("ERROR: while decoding base64 from TTN message: %v", err)
		return
	}

	// Bounds check: need at least 1 byte to trim trailing null
	if len(data) == 0 {
		log.Printf("ERROR: empty payload from TTN message")
		return
	}
	// Strip trailing null byte only if present.
	// Older firmwares sent strlen+1 (with null); current firmware sends strlen (no null).
	if data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}

	stringSlice := strings.Split(string(data), ",")
	if len(stringSlice) < 3 {
		log.Printf("ERROR: payload too short, expected at least 3 fields, got %d: %q", len(stringSlice), string(data))
		return
	}

	sensoreID := stringSlice[0]
	NumberOfvalues := (len(stringSlice) - 1) / 2
	var shift = 1
	for i := 0; i < NumberOfvalues; i++ {
		if shift+1 >= len(stringSlice) {
			log.Printf("ERROR: payload index out of bounds at shift=%d, len=%d", shift, len(stringSlice))
			break
		}

		typeID, err := strconv.Atoi(stringSlice[shift])
		if err != nil {
			log.Printf("ERROR: invalid typeID %q: %v", stringSlice[shift], err)
			shift += 2
			continue
		}

		// type 2 = firmware version string (not a float — handle separately)
		if typeID == 2 {
			version := stringSlice[shift+1]
			shift += 2
			found := false
			for _, c := range clients {
				if c.UUID == sensoreID {
					found = true
					break
				}
			}
			if !found {
				log.Printf("WARN: unknown sensor %q, skipping version update", sensoreID)
				continue
			}
			if version == "" {
				log.Printf("WARN: sensor %q reported empty firmware version, skipping", sensoreID)
				continue
			}
			patchFirmwareVersion(sensoreID, version)
			continue
		}

		value, err := strconv.ParseFloat(stringSlice[shift+1], 64)
		if err != nil {
			log.Printf("ERROR: invalid value %q: %v", stringSlice[shift+1], err)
			shift += 2
			continue
		}
		shift = shift + 2
		clientOfMessage := structs2.Clients{}
		densityData := structs2.DensityData{
			SensorID: sensoreID,
			Value:    value,
		}
		found := false
		for _, c := range clients {
			if c.UUID == densityData.SensorID {
				clientOfMessage = c
				found = true
			}
		}
		if !found {
			log.Printf("WARN: unknown sensor %q, skipping", sensoreID)
			continue
		}
		var DataWithClient structs2.DensityDataWithClient
		switch typeID {
		case 0:
			DataWithClient.DataType = "densityData"
		case 1:
			DataWithClient.DataType = "batteryData"
		}

		DataWithClient.Client = clientOfMessage
		DataWithClient.Data = densityData
		encodedData, err := json.Marshal(DataWithClient)
		if err != nil {
			log.Printf("ERROR: failed to marshal sensor data: %v", err)
			shift += 2
			continue
		}
		_, err = restyClient.R().
			SetHeader(internalKeyHeader, Misc2.GetInternalAPIKey()).
			SetBody(encodedData).
			Post(fmt.Sprintf("%s/internal/sensor-data", Misc2.GetBackendURL()))
		if err != nil {
			log.Printf("ERROR: failed to send data to backend: %v", err)
		}
	}
}

func patchFirmwareVersion(uuid, version string) {
	body := fmt.Sprintf(`{"version":%q}`, version)
	_, err := resty.New().R().
		SetHeader(internalKeyHeader, Misc2.GetInternalAPIKey()).
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Patch(fmt.Sprintf("%s/internal/sensors/%s/firmware-version", Misc2.GetBackendURL(), uuid))
	if err != nil {
		log.Printf("ERROR: failed to patch firmware version for %s: %v", uuid, err)
	}
}

func main() {
	Misc2.StartUp()
	log.Println("STARTING AGGREGATOR")
	broker, clientID, topic, username, password, _ := Misc2.SetupVars()
	opts := mqtt.NewClientOptions()
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.SetUsername(username)
	opts.SetPassword(password)
	mqttClient := Mqtt.StartMqtttConnection(broker, clientID, opts)
	sub(mqttClient, topic)

	signals()
}

func sub(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	log.Printf("Subscribed to topic: %s", topic)
}

func signals() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		log.Printf("Received signal: %s", sig)
		done <- true
	}()
	<-done
	log.Println("Exiting")
}
