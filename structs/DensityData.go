package structs

import "github.com/google/uuid"

type DensityData struct {
	SensorID uuid.UUID
	Value    float64
}
