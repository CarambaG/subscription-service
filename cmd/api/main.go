package main

import (
	"fmt"
	"os"

	"github.com/CarambaG/subscription-service/internal/config"
)

func main() {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "config/config.yaml"
	}
	if _, err := config.Load(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
