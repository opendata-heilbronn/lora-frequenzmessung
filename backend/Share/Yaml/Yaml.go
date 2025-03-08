package Yaml

import (
	"bytes"
	"github.com/opendata-heilbronn/lora-frequenzmessung/structs"
	"github.com/spf13/viper"
	"os"
)

func LoadYaml() []structs.Clients {
	viper.SetConfigType("yaml")
	dat, err := os.ReadFile("backend/Aggregator/clients.yml")
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
