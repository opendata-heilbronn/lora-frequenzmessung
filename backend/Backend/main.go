package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/Misc"
	structs2 "github.com/opendata-heilbronn/lora-frequenzmessung/structs"
	"os"
	"time"
)

func main() {
	Misc.StartUp()
	println("STARTING Backend")

	app := fiber.New()
	ctx := context.Background()
	DBDns := Misc.GetDBDsn()
	conn, err := pgx.Connect(ctx, DBDns)
	defer conn.Close(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, " database: %v\n", DBDns)
		fmt.Println("------------------------------------")
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Define a route for the GET method on the root path '/'
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

	// Start the server on port 3000
	err = app.Listen(":3001")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to start server: %v\n", err)
		return
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
