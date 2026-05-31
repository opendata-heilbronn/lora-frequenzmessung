package misc

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func StartUp() {
	printCFH()
	loadEnv()
}

func loadEnv() {
	// if dotenv file is not present
	if _, err := os.Stat(".env"); errors.Is(err, os.ErrNotExist) {
		println("No .env file found")
		return
	}

	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		os.Exit(1)
	}
}
