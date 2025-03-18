package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	BasicAuth         string
	ClickhouseURL     string
	FlushInterval     time.Duration
	ListenerPort      string
	LogDebug          bool
	Logger            *slog.Logger
	ServiceName       string
	ServiceVersion    string
	TraceMaxBatchSize int
	TraceSampleRate   float64
}

func LoadConfig() (*Config, error) {
	// Defaults set are for production use.
	// Configure to your liking for development/testing
	config := &Config{
		FlushInterval:     time.Second * 5,
		ListenerPort:      "7123",
		LogDebug:          false,
		ServiceName:       "chproxy",
		ServiceVersion:    "1.3.1",
		TraceMaxBatchSize: 512,
		TraceSampleRate:   0.25, // Sample 25%
	}

	// Generic logger
	config.Logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

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

	if sampleRateStr := os.Getenv("OTEL_TRACE_SAMPLE_RATE"); sampleRateStr != "" {
		sampleRate, err := strconv.ParseFloat(sampleRateStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid TRACE_SAMPLE_RATE: %w", err)
		}
		if sampleRate < 0.0 || sampleRate > 1.0 {
			return nil, fmt.Errorf("OTEL_TRACE_SAMPLE_RATE must be between 0.0 and 1.0")
		}
		config.TraceSampleRate = sampleRate
	}

	return config, nil
}
