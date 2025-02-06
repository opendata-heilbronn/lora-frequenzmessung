package main

import (
	"fmt"
	"github.com/caarlos0/env/v11"
	"github.com/google/uuid"
	"github.com/opendata-heilbronn/lora-frequenzmessung/structs"
	"math/rand"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type config struct {
	BrokenHost string `env:"BROKEN_HOST"`
	BrokenPort string `env:"BROKEN_PORT"`
	ClientID   string `env:"CLIENTID"`
	Topic      string `env:"TOPIC"`
}

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
	setupVars()
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	for {
		uuid, err := uuid.Parse("e529df2e-03ef-4ae1-9b6f-f4b894f9b0db")
		if err != nil {
			panic(err)
		}
		message := structs.DensityData{
			SensorID: uuid,
			Value:    float64(rand.Intn(100)),
		}

		token := client.Publish(topic, 0, false, message)
		token.Wait()
		fmt.Printf("Published message: %s\n", message)
		time.Sleep(1 * time.Second)
	}
}

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}
func setupVars() {
	var cfg config
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}
	cfg, err = env.ParseAs[config]()
	if err != nil {
		panic(err)
	}
	broker = fmt.Sprintf("tcp://%s:%s", cfg.BrokenHost, cfg.BrokenPort)
	clientID = cfg.ClientID
}
