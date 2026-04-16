package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/database"
	"github.com/opendata-heilbronn/lora-frequenzmessung/Share/pwhash"
)

func main() {
	loadEnv()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		fatalf("DB_DSN environment variable not set")
	}

	db, err := database.Connect(database.Config{URI: dsn, MaxOpenConns: 1})
	if err != nil {
		fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	var username, password string

	switch len(os.Args) {
	case 1:
		username, err = selectUserInteractive(ctx, db)
		if err != nil {
			fatalf("%v", err)
		}
	case 2:
		username = os.Args[1]
	case 3:
		username = os.Args[1]
		password = os.Args[2]
	default:
		fatalf("usage: reset-password [username [new-password]]")
	}

	var userID string
	err = db.QueryRowContext(ctx, `SELECT id FROM users WHERE username = $1`, username).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		fatalf("user %q not found", username)
	}
	if err != nil {
		fatalf("failed to look up user: %v", err)
	}

	if password == "" {
		password = generatePassword()
	}

	hash, err := pwhash.Create(password)
	if err != nil {
		fatalf("failed to hash password: %v", err)
	}

	_, err = db.ExecContext(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, hash, userID)
	if err != nil {
		fatalf("failed to update password: %v", err)
	}

	fmt.Printf("Password for %q reset to: %s\n", username, password)
}

func selectUserInteractive(ctx context.Context, db *sql.DB) (string, error) {
	rows, err := db.QueryContext(ctx, `SELECT username FROM users ORDER BY username`)
	if err != nil {
		return "", fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return "", err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(users) == 0 {
		return "", errors.New("no users found in database")
	}

	fmt.Println("Select a user:")
	for i, u := range users {
		fmt.Printf("  %d) %s\n", i+1, u)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("Enter number (1-%d): ", len(users))
		if !scanner.Scan() {
			return "", errors.New("no input provided")
		}
		input := strings.TrimSpace(scanner.Text())
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > len(users) {
			fmt.Printf("  Please enter a number between 1 and %d\n", len(users))
			continue
		}
		return users[n-1], nil
	}
}

func loadEnv() {
	if _, err := os.Stat(".env"); errors.Is(err, os.ErrNotExist) {
		return
	}
	if err := godotenv.Load(); err != nil {
		fatalf("error loading .env file: %v", err)
	}
}

func generatePassword() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fatalf("failed to generate random password: %v", err)
	}
	return base64.RawStdEncoding.EncodeToString(b)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
