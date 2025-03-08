package structs

type DensityData struct {
	SensorID string
	Value    float64
}
type DensityDataWithClient struct {
	Data     DensityData
	Client   Clients
	DataType string
}
