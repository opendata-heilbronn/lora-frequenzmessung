package main

import (
	"bytes"
	"fmt"
	"github.com/opendata-heilbronn/lora-frequenzmessung/structs"
	"github.com/spf13/viper"
	"os"
)

func main() {
	clients := loadYaml()
	fmt.Println(clients[0].Name)
	for _, client := range clients {
		fmt.Println(client.Name)
	}
}
func loadYaml() []structs.Clients {
	viper.SetConfigType("yaml")
	dat, err := os.ReadFile("Backend/clients.yml")
	if err != nil {
		panic(err)
	}
	viper.ReadConfig(bytes.NewBuffer(dat))
	var clients []structs.Clients
	err = viper.UnmarshalKey("clients", &clients)
	if err != nil {
		panic(err)
	}
	return clients
}
