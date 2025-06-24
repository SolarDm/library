package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type (
	Config struct {
		GRPC
		PG
		Outbox
	}

	GRPC struct {
		Port        string `env:"GRPC_PORT"`
		GatewayPort string `env:"GRPC_GATEWAY_PORT"`
	}

	PG struct {
		URL      string
		Host     string `env:"POSTGRES_HOST"`
		Port     string `env:"POSTGRES_PORT"`
		DB       string `env:"POSTGRES_DB"`
		User     string `env:"POSTGRES_USER"`
		Password string `env:"POSTGRES_PASSWORD"`
		MaxConn  string `env:"POSTGRES_MAX_CONN"`
	}

	Outbox struct {
		Enabled         bool          `env:"OUTBOX_ENABLED"`
		Workers         int           `env:"OUTBOX_WORKERS"`
		BatchSize       int           `env:"OUTBOX_BATCH_SIZE"`
		WaitTimeMS      time.Duration `env:"OUTBOX_WAIT_TIME_MS"`
		InProgressTTLMS time.Duration `env:"OUTBOX_IN_PROGRESS_TTL_MS"`
		AuthorSendURL   string        `env:"OUTBOX_AUTHOR_SEND_URL"`
		BookSendURL     string        `env:"OUTBOX_BOOK_SEND_URL"`
	}
)

func getOrDefault(envName string, defaultValue string) string {
	if val, exist := os.LookupEnv(envName); exist {
		return val
	}
	return defaultValue
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	cfg.GRPC.Port = getOrDefault("GRPC_PORT", "9090")
	cfg.GRPC.GatewayPort = getOrDefault("GRPC_GATEWAY_PORT", "8080")

	cfg.PG.Host = getOrDefault("POSTGRES_HOST", "127.0.0.1")
	cfg.PG.Port = getOrDefault("POSTGRES_PORT", "5432")
	cfg.PG.DB = getOrDefault("POSTGRES_DB", "library")
	cfg.PG.User = getOrDefault("POSTGRES_USER", "dima")
	cfg.PG.Password = getOrDefault("POSTGRES_PASSWORD", "1234")
	cfg.PG.MaxConn = getOrDefault("POSTGRES_MAX_CONN", "10")

	pgURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.PG.User, cfg.PG.Password),
		Host:     net.JoinHostPort(cfg.PG.Host, cfg.PG.Port),
		Path:     cfg.PG.DB,
		RawQuery: "sslmode=disable&pool_max_conns=" + cfg.PG.MaxConn,
	}

	cfg.PG.URL = pgURL.String()

	var err error
	cfg.Outbox.Enabled, err = strconv.ParseBool(getOrDefault("OUTBOX_ENABLED", "false"))

	if err != nil {
		return nil, fmt.Errorf("error while parsing OUTBOX_ENABLED: %w", err)
	}

	if cfg.Outbox.Enabled {
		cfg.Outbox.Workers, err = strconv.Atoi(os.Getenv("OUTBOX_WORKERS"))

		if err != nil {
			return nil, fmt.Errorf("error while parsing OUTBOX_WORKERS: %w", err)
		}

		cfg.Outbox.BatchSize, err = strconv.Atoi(os.Getenv("OUTBOX_BATCH_SIZE"))

		if err != nil {
			return nil, fmt.Errorf("error while parsing OUTBOX_BATCH_SIZE: %w", err)
		}

		waitTime, err := strconv.Atoi(os.Getenv("OUTBOX_WAIT_TIME_MS"))

		if err != nil {
			return nil, fmt.Errorf("error while parsing OUTBOX_WAIT_TIME_MS: %w", err)
		}

		cfg.Outbox.WaitTimeMS = time.Duration(waitTime) * time.Millisecond

		inProgressTTL, err := strconv.Atoi(os.Getenv("OUTBOX_IN_PROGRESS_TTL_MS"))

		if err != nil {
			return nil, fmt.Errorf("error while parsing OUTBOX_IN_PROGRESS_TTL_MS: %w", err)
		}

		cfg.Outbox.InProgressTTLMS = time.Duration(inProgressTTL) * time.Millisecond

		cfg.Outbox.AuthorSendURL = os.Getenv("OUTBOX_AUTHOR_SEND_URL")
		cfg.Outbox.BookSendURL = os.Getenv("OUTBOX_BOOK_SEND_URL")
	}

	return cfg, nil
}
