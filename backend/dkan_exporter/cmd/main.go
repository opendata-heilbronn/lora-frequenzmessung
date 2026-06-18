package main

import (
	"fmt"
	"os"

	"codeberg.org/cfhn/lorax.git/backend/dkan_exporter/internal/config"
	"codeberg.org/cfhn/lorax.git/backend/dkan_exporter/internal/gateways/dkan"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	cfg, err := config.Parse()
	if err != nil {
		return err
	}

	fmt.Printf("running dkan test\n")

	dkanClient := dkan.NewClient("https://opendata.heilbronn.de", cfg.DKANUser, cfg.DKANPassword)
	dkanClient.GetRevisions("passantenzählung")
	fmt.Printf("finished running dkan test\n")

	return nil
}
