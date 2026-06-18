package dkan

// Frequency represents the fast-moving fact data.
// This generates the "Historical Pedestrian Counts" CSV.
type Frequency struct {
	Timestamp   string `csv:"timestamp"`
	SensorID    string `csv:"sensor_id"`
	PeopleCount int    `csv:"people_count"`
}
