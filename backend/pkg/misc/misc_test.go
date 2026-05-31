package misc_test

import (
	"os"
	"testing"

	"codeberg.org/cfhn/lorax.git/backend/pkg/misc"
)

func TestGetDBDns(t *testing.T) {
	t.Setenv("DB_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")

	got := misc.GetDBDsn()
	if got != os.Getenv("DB_DSN") {
		t.Errorf("GetDBDns() = %v, want %v", got, os.Getenv("DB_DSN"))
	}
}

func TestGetBackendURL(t *testing.T) {
	t.Setenv("BACKEND_URL", "http://localhost:3001")

	got := misc.GetBackendURL()
	if got != os.Getenv("BACKEND_URL") {
		t.Errorf("GetBackendURL() = %v, want %v", got, os.Getenv("BACKEND_URL"))
	}
}

func TestSetupVars(t *testing.T) {
	t.Setenv("BROKER_HOST", "localhost")
	t.Setenv("BROKER_PORT", "1883")
	t.Setenv("CLIENTID", "clientID")
	t.Setenv("TOPIC", "topic")
	t.Setenv("MQTT_USERNAME", "username")
	t.Setenv("MQTT_PASSWORD", "password")
	t.Setenv("DB_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")

	broker, clientID, topic, username, password, dbDsn := misc.SetupVars()
	if broker != "tcp://localhost:1883" {
		t.Errorf("SetupVars() broker = %v, want %v", broker, "tcp://localhost:1883")
	}

	if clientID != "clientID" {
		t.Errorf("SetupVars() clientID = %v, want %v", clientID, "clientID")
	}

	if topic != "topic" {
		t.Errorf("SetupVars() topic = %v, want %v", topic, "topic")
	}

	if username != "username" {
		t.Errorf("SetupVars() username = %v, want %v", username, "username")
	}

	if password != "password" {
		t.Errorf("SetupVars() password = %v, want %v", password, "password")
	}

	if dbDsn != "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" {
		t.Errorf("SetupVars() dbDsn = %v, want %v", dbDsn, "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	}
}
