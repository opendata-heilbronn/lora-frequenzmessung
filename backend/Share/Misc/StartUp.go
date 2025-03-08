package Misc

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

func StartUp() {
	printCFH()
	loadEnv()
}
func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
		os.Exit(1)
	}
}
