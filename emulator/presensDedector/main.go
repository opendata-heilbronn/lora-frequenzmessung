package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Mqtt"
	"github.com/opendata-heilbronn/lora-frequenzmessung/structs"
	"math/rand"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var broker string
var clientID string
var topic string

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to MQTT Broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection lost: %v", err)
}

func main() {
	broker, clientID, topic := Misc.SetupVars()
	opts := mqtt.NewClientOptions()
	client := Mqtt.StartMqtttConnection(broker, clientID, opts)

	for {
		uuid, err := uuid.Parse("e529df2e-03ef-4ae1-9b6f-f4b894f9b0db")
		if err != nil {
			panic(err)
		}
		message := structs.DensityData{
			SensorID: uuid,
			Value:    float64(rand.Intn(100)),
		}
		messageString, err := json.Marshal(message)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		token := client.Publish(topic, 0, false, messageString)
		token.Wait()
		fmt.Printf("Published message: %s\n", messageString)
		time.Sleep(time.Second)
	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
