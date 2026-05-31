package dkan

import (
	"fmt"
	"os"
	"time"

	"github.com/gocarina/gocsv"
)

// Sensor represents the dictionary of your physical sensors.
// This generates the "Locations and Metadata" CSV.
type Sensor struct {
	SensorID            string    `csv:"sensor_id"`
	Description         string    `csv:"description"`
	Latitude            float64   `csv:"latitude"`
	Longitude           float64   `csv:"longitude"`
	LocationDescription string    `csv:"location_description"`
	CreatedAt           time.Time `csv:"created_at"`
}

// ExportMetadataToCSV takes a slice of SensorMetadata and writes it to a CSV file.
// If the file does not exist, it creates it. If it does, it overwrites it.
func ExportMetadataToCSV(metadata []Sensor, filepath string) error {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open metadata file %s: %w", filepath, err)
	}
	defer file.Close()

	if err := gocsv.MarshalFile(&metadata, file); err != nil {
		return fmt.Errorf("failed to marshal metadata to CSV: %w", err)
	}

	return nil
}
