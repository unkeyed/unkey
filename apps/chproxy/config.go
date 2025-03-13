package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

type Config struct {
	LogDebug       bool
	Logger         *slog.Logger
	BasicAuth      string
	ClickhouseURL  string
	FlushInterval  time.Duration
	ListenerPort   string
	MaxBatchSize   int
	MaxBufferSize  int
	ServiceName    string
	ServiceVersion string
}

func LoadConfig() (*Config, error) {
	// New config with defaults
	config := &Config{
		LogDebug:       false,
		FlushInterval:  time.Second * 5,
		ListenerPort:   "7123",
		MaxBatchSize:   10000,
		MaxBufferSize:  50000,
		ServiceName:    "chproxy",
		ServiceVersion: "1.2.0",
	}

	config.ClickhouseURL = os.Getenv("CLICKHOUSE_URL")
	if config.ClickhouseURL == "" {
		return nil, fmt.Errorf("CLICKHOUSE_URL must be defined")
	}

	config.BasicAuth = os.Getenv("BASIC_AUTH")
	if config.BasicAuth == "" {
		return nil, fmt.Errorf("BASIC_AUTH must be defined")
	}

	if debug := os.Getenv("OTEL_EXPORTER_LOG_DEBUG"); debug == "true" {
		config.LogDebug = true
	}

	if port := os.Getenv("PORT"); port != "" {
		config.ListenerPort = port
	}

	return config, nil
}
