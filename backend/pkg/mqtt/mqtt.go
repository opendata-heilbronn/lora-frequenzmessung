package mqtt

import (
	"fmt"
	"math/rand"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func StartMqtttConnection(broker string, clientID string, opts *mqtt.ClientOptions) mqtt.Client {
	clientID = fmt.Sprintf("%s-%d", clientID, rand.Int())

	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println("Error connecting to MQTT Broker")
		panic(token.Error())
	}

	return client
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Connected to MQTT Broker")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connection lost: %v", err)
	os.Exit(1)
}
