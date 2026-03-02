package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort   string
	PGHost     string
	PGPort     string
	PGUser     string
	PGPassword string
	PGDB       string
	PGSSLMode  string
}

func Load(envName string) (*Config, error) {
	err := godotenv.Load(envName)
	if err != nil {
		log.Fatal("Error loading .env file")
		return nil, err
	}

	return &Config{
		HTTPPort:   os.Getenv("HTTP_PORT"),
		PGHost:     os.Getenv("POSTGRES_HOST"),
		PGPort:     os.Getenv("POSTGRES_PORT"),
		PGUser:     os.Getenv("POSTGRES_USER"),
		PGPassword: os.Getenv("POSTGRES_PASSWORD"),
		PGDB:       os.Getenv("POSTGRES_DB"),
		PGSSLMode:  os.Getenv("POSTGRES_SSLMODE"),
	}, nil
}

func (c Config) PostgresDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.PGUser, c.PGPassword, c.PGHost, c.PGPort, c.PGDB, c.PGSSLMode,
	)
}
