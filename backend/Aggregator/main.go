package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-resty/resty/v2"
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

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	//todo add checker identify data type
	// tod hint name use the name of the sensores
	fmt.Println("Received message: ", msg.MessageID())
	var ttnMessage structs2.TtnMessage
	resty := resty.New()
	clients := Yaml.LoadYaml()

	err := json.Unmarshal(msg.Payload(), &ttnMessage)
	if err != nil {
		log.Fatalf("Unable to marshal JSON due to %s", err)
	}
	data, err := base64.StdEncoding.DecodeString(ttnMessage.UplinkMessage.FrmPayload)
	if err != nil {
		log.Fatal("while decoding bas64 from TTN Message:", err)
	}
	data = data[:len(data)-1] //remove last byte as it is null
	stringSlice := strings.Split(string(data), ",")
	sensoreID := stringSlice[0]
	value, _ := strconv.ParseFloat(stringSlice[1], 64)
	clientOfMessage := structs2.Clients{}
	densityData := structs2.DensityData{
		SensorID: sensoreID,
		Value:    value,
	}
	found := false
	for _, client := range clients {
		if client.UUID == densityData.SensorID {
			clientOfMessage = client
			found = true
		}
	}

	if !found {
		return
	}
	var DataWithClient structs2.DensityDataWithClient

	DataWithClient.Client = clientOfMessage
	DataWithClient.Data = densityData
	DataWithClient.DataType = "densityData"
	encodedData, _ := json.Marshal(DataWithClient)
	fmt.Println(string(encodedData))
	resty.R().SetBody(encodedData).Post(fmt.Sprintf("%s/add-sensor-data", Misc2.GetBackendURL()))
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
