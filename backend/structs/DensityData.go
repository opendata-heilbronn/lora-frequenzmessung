package structs

type DensityData struct {
	SensorID string
	Value    float64
}

type BatteryChargeData struct {
	SensorID string
	Value    float64
}

type DensityDataWithClient struct {
	Data     DensityData
	Client   Clients
	DataType string
}

type BatteryChargeDataWithClient struct {
	Data     BatteryChargeData
	Client   Clients
	DataType string
}
