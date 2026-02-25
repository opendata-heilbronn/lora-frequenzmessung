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

func loadSensors() ([]structs2.Clients, error) {
	sensorCache.Lock()
	defer sensorCache.Unlock()

	if time.Since(sensorCache.fetchedAt) < sensorCacheTTL && sensorCache.sensors != nil {
		return sensorCache.sensors, nil
	}

	backendURL := Misc2.GetBackendURL()
	resp, err := resty.New().R().
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%s/api/sensors", backendURL))
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
	fmt.Println("Received message: ", msg.MessageID())
	var ttnMessage structs2.TtnMessage
	restyClient := resty.New()

	clients, err := loadSensors()
	if err != nil {
		fmt.Println("Failed to load sensors:", err)
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
	data = data[:len(data)-1] //remove last byte as it is null

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
			return
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
		encodedData, _ := json.Marshal(DataWithClient)
		fmt.Println(string(encodedData))
		_, err = restyClient.R().SetBody(encodedData).Post(fmt.Sprintf("%s/add-sensor-data", Misc2.GetBackendURL()))
		if err != nil {
			fmt.Println("cant send data to Backend due to: ")
			fmt.Println(err)
		}
	}
}

func main() {
	Misc2.StartUp()
	println("STARTING AGGREGATOR")
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
	fmt.Printf("Subscribed to topic: %s", topic)
	fmt.Println()
}

func signals() {
	fmt.Println("Starting signal handler")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()
	fmt.Println("awaiting signal")
	<-done
	fmt.Println("exiting")
}
