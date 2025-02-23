package structs

import (
	"github.com/google/uuid"
)

type DensityData struct {
	SensorID uuid.UUID
	Value    float64
}
type DensityDataWithClient struct {
	Data     DensityData
	Client   Clients
	DataType string
}
