package main

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Mqtt"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Yaml"
	"github.com/opendata-heilbronn/lora-frequenzmessung/structs"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var broker string
var clientID string
var topic string

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var messageDens structs.DensityData

	err := json.Unmarshal(msg.Payload(), &messageDens)
	if err != nil {
		log.Fatalf("Unable to marshal JSON due to %s", err)
	}
	clients := Yaml.LoadYaml()
	//for client in clients:
	clientOfMessage := structs.Clients{}
	found := false
	for _, client := range clients {
		if client.UUID == messageDens.SensorID.String() {
			clientOfMessage = client
			found = true
		}
	}
	if !found {
		return
	}

	fmt.Printf("Sensore ID: %s, Value: %f, Name: %s\n", messageDens.SensorID, messageDens.Value, clientOfMessage.Name)
}

func main() {
	//clientsData := Yaml.LoadYaml()
	broker, clientID, topic := Misc.SetupVars()
	opts := mqtt.NewClientOptions()
	opts.SetDefaultPublishHandler(messagePubHandler)
	mqttClient := Mqtt.StartMqtttConnection(broker, clientID, opts)
	sub(mqttClient, topic)

	signals()

}
func sub(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic: %s", topic)
}
func signals() {
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
