package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      App      `yaml:"app"`
	Database Database `yaml:"database"`
	Log      Log      `yaml:"log"`
}

type App struct {
	Address         string        `yaml:"address"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type Database struct {
	URL            string        `yaml:"url"`
	MaxConns       int32         `yaml:"max_conns"`
	MinConns       int32         `yaml:"min_conns"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
}

type Log struct {
	Level string `yaml:"level"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	if err := applyEnvironment(&cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.App.Address == "" {
		return errors.New("app address is required")
	}
	if c.App.ShutdownTimeout <= 0 {
		return errors.New("app shutdown timeout must be positive")
	}
	if c.Database.URL == "" {
		return errors.New("database URL is required")
	}
	if c.Database.MaxConns <= 0 {
		return errors.New("database max connections must be positive")
	}
	if c.Database.MinConns < 0 || c.Database.MinConns > c.Database.MaxConns {
		return errors.New("database min connections must be between zero and max connections")
	}
	if c.Database.ConnectTimeout <= 0 {
		return errors.New("database connect timeout must be positive")
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unsupported log level %q", c.Log.Level)
	}
	return nil
}

func applyEnvironment(cfg *Config) error {
	setString("APP_ADDRESS", &cfg.App.Address)
	setString("DATABASE_URL", &cfg.Database.URL)
	setString("LOG_LEVEL", &cfg.Log.Level)

	if err := setDuration("APP_SHUTDOWN_TIMEOUT", &cfg.App.ShutdownTimeout); err != nil {
		return err
	}
	if err := setDuration("DATABASE_CONNECT_TIMEOUT", &cfg.Database.ConnectTimeout); err != nil {
		return err
	}
	if err := setInt32("DATABASE_MAX_CONNS", &cfg.Database.MaxConns); err != nil {
		return err
	}
	if err := setInt32("DATABASE_MIN_CONNS", &cfg.Database.MinConns); err != nil {
		return err
	}
	return nil
}

func setString(name string, target *string) {
	if value, ok := os.LookupEnv(name); ok {
		*target = value
	}
}

func setDuration(name string, target *time.Duration) error {
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	*target = parsed
	return nil
}

func setInt32(name string, target *int32) error {
	value, ok := os.LookupEnv(name)
	if !ok {
		return nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	*target = int32(parsed)
	return nil
}
