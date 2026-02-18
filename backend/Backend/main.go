package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/jackc/pgx/v5"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
	structs2 "github.com/opendata-heilbronn/lora-frequenzmessung/structs"
)

func main() {
	Misc.StartUp()
	println("STARTING Backend")

	app := fiber.New()
	ctx := context.Background()
	DBDns := Misc.GetDBDsn()
	conn, err := pgx.Connect(ctx, DBDns)
	if err != nil {
		fmt.Fprintf(os.Stderr, " database: %v\n", DBDns)
		fmt.Println("------------------------------------")
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// CORS — allow frontend dev server and production frontend
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://localhost:8080", "*"},
		AllowMethods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	// Existing route
	app.Post("/add-sensor-data", func(c fiber.Ctx) error {
		p := new(structs2.DensityDataWithClient)
		err := c.AutoFormat(p)
		if err != nil {
			return err
		}
		err = json.Unmarshal(c.Body(), p)
		if err != nil {
			return err
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

	// Sensor CRUD
	app.Get("/api/sensors", getSensors(conn, ctx))
	app.Post("/api/sensors", createSensor(conn, ctx))
	app.Delete("/api/sensors/:uuid", deleteSensor(conn, ctx))
	app.Post("/api/sensors/:uuid/register-ttn", registerTTN(conn, ctx))

	// Firmware build & serving
	app.Post("/api/sensors/:uuid/build-firmware", buildFirmware(conn, ctx))
	app.Get("/api/sensors/:uuid/build-status", getBuildStatus())
	app.Get("/api/sensors/:uuid/manifest.json", getFirmwareManifest(conn, ctx))
	app.Get("/api/sensors/:uuid/firmware.bin", getFirmwareBin())

	err = app.Listen(":3001")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start server: %v\n", err)
	}
}

func sendData(conn *pgx.Conn, ctx context.Context, uuid string, sensorName string, longitude float64, latitude float64, value float64, sensorType string) {
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
