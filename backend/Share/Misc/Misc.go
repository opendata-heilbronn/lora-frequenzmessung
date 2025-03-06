package Misc

import (
	"fmt"
	"github.com/caarlos0/env/v11"
)

type config struct {
	BrokenHost string `env:"BROKEN_HOST"`
	BrokenPort string `env:"BROKEN_PORT"`
	ClientID   string `env:"CLIENTID"`
	Topic      string `env:"TOPIC"`
}

func SetupVars() (string, string, string) {
	var cfg config
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}
	cfg, err = env.ParseAs[config]()
	if err != nil {
		panic(err)
	}
	broker := fmt.Sprintf("tcp://%s:%s", cfg.BrokenHost, cfg.BrokenPort)
	clientID := cfg.ClientID
	topic := cfg.Topic
	return broker, clientID, topic
}
