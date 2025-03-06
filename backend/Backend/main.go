package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/opendata-heilbronn/lora-frequenzmessung/backend-/structs"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	app := fiber.New()
	ctx := context.Background()
	connStr := "postgres://timescaledb:password@localhost:5432/postgres"
	conn, err := pgx.Connect(ctx, connStr)
	defer conn.Close(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Define a route for the GET method on the root path '/'
	app.Post("/add-sensor-data", func(c fiber.Ctx) error {
		p := new(structs.DensityDataWithClient)
		err := c.AutoFormat(p)
		if err != nil {
			return err
		}
		err = json.Unmarshal(c.Body(), p)
		if err != nil {
			return err
		}

		//todo do add migration
		if err != nil {
			panic(err)
		}
		sendData(
			conn,
			ctx,
			p.Data.SensorID,
			p.Client.Name,
			p.Client.Longitude,
			p.Client.Latitude,
			p.Data.Value,
			p.DataType)

		return c.Status(fiber.StatusAccepted).SendString("Message accepted")
	})

	// Start the server on port 3000
	err = app.Listen(":3001")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start server: %v\n", err)
		return
	}

}
func sendData(conn *pgx.Conn, ctx context.Context, uuid uuid.UUID, sensorName string, longitude float64, latitude float64, value float64, sensorType string) {
	t := time.Now()
	queryInsertMetadata := `INSERT INTO sensor_data (
                         sensor_id,
                         name,
                         time,
                         longitude,
                         latitude,
                         value,
                         type
                         ) VALUES ($1, $2,$3,$4,$5,$6,$7);`

	_, err := conn.Exec(ctx, queryInsertMetadata, uuid, sensorName, t, longitude, latitude, value, sensorType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to insert data into database: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Inserted sensor (%s, %v) into database \n", sensorName, value)
}
