package main

import (
	"encoding/json"
	"fmt"
	Misc2 "github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
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
	Misc2.StartUp()
	broker, clientID, topic, _, _, _ := Misc2.SetupVars()
	opts := mqtt.NewClientOptions()
	client := Mqtt.StartMqtttConnection(broker, clientID, opts)

	for {
		uuid := "f4b894f9b0db"
		message := structs.DensityData{
			SensorID: uuid,
			Value:    float64(rand.Intn(100)),
		}
		messageString, err := json.Marshal(message)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		token := client.Publish(topic, 0, true, messageString)
		token.Wait()
		fmt.Printf("Published message: %s\n", messageString)
		time.Sleep(time.Second)
	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
