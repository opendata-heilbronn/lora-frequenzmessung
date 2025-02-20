package main

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Mqtt"
	"os"
	"os/signal"
	"syscall"
)

var broker string
var clientID string
var topic string

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
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
