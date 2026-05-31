package dkan

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"
)

func GenerateCSV(sensors []Sensor) ([]byte, error) {
	var buf bytes.Buffer

	writer := csv.NewWriter(&buf)

	header := []string{"sensor_id", "description", "latitude", "longitude", "created_at"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("error writing header: %w", err)
	}

	for _, s := range sensors {
		record := []string{
			s.SensorID,
			s.Description,
			// FormatFloat converts the float64 to a string.
			// 'f' means no exponents, -1 means print all necessary digits.
			strconv.FormatFloat(s.Latitude, 'f', -1, 64),
			strconv.FormatFloat(s.Longitude, 'f', -1, 64),
			// Format the time back to a string (using RFC3339 to match earlier)
			s.CreatedAt.Format(time.RFC3339),
		}

		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("error writing record for sensor %s: %w", s.SensorID, err)
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("error flushing csv writer: %w", err)
	}

	return buf.Bytes(), nil
}
