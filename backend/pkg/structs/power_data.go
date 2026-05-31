package structs

import "github.com/google/uuid"

type PowerData struct {
	SensorID uuid.UUID
	Value    float64
}
