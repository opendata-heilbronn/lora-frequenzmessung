package Misc

import (
	"fmt"
	"github.com/caarlos0/env/v11"
)

type config struct {
	BrokenHost    string `env:"BROKEN_HOST"`
	BrokenPort    string `env:"BROKEN_PORT"`
	ClientID      string `env:"CLIENTID"`
	Topic         string `env:"TOPIC"`
	MQTT_username string `env:"MQTT_USERNAME"`
	MQTT_password string `env:"MQTT_PASSWORD"`
	DBdsn         string `env:"DB_DSN"`
}
type backendURL struct {
	backendURL string `env:"BACKEND_URL"`
}

func GetBackendURL() string {
	var cfg backendURL
	err := env.Parse(&cfg)
	if err != nil {
		panic(err)
	}
	cfg, err = env.ParseAs[backendURL]()
	return cfg.backendURL
}

func SetupVars() (string, string, string, string, string, string) {
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
	username := cfg.MQTT_username
	password := cfg.MQTT_password
	dbDsn := cfg.DBdsn
	return broker, clientID, topic, username, password, dbDsn
}
