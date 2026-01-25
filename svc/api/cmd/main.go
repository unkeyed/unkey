package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/api"
)

func main() {
	cfg := api.Config{
		InstanceID:              getEnv("INSTANCE_ID", "api-subprocess"),
		Platform:                getEnv("PLATFORM", "test"),
		Image:                   getEnv("IMAGE", "test"),
		HttpPort:                getEnvInt("HTTP_PORT", 7070),
		Listener:                nil, // Will be created from HttpPort
		Region:                  getEnv("REGION", "test"),
		RedisUrl:                getEnvRequired("REDIS_URL"),
		TestMode:                getEnvBool("TEST_MODE", true),
		ClickhouseURL:           getEnv("CLICKHOUSE_URL", ""),
		ClickhouseAnalyticsURL:  getEnv("CLICKHOUSE_ANALYTICS_URL", ""),
		DatabasePrimary:         getEnvRequired("DATABASE_PRIMARY"),
		DatabaseReadonlyReplica: getEnv("DATABASE_READONLY_REPLICA", ""),
		OtelEnabled:             getEnvBool("OTEL_ENABLED", false),
		OtelTraceSamplingRate:   0.0,
		PrometheusPort:          0,
		Clock:                   clock.New(),
		TLSConfig:               nil,
		VaultMasterKeys:         getEnvSlice("VAULT_MASTER_KEYS", ","),
		VaultS3:                 nil,
		KafkaBrokers:            getEnvSlice("KAFKA_BROKERS", ","),
		CacheInvalidationTopic:  getEnv("CACHE_INVALIDATION_TOPIC", ""),
		ChproxyToken:            getEnv("CHPROXY_TOKEN", ""),
		CtrlURL:                 getEnv("CTRL_URL", "http://ctrl:7091"),
		CtrlToken:               getEnv("CTRL_TOKEN", "your-local-dev-key"),
		PprofEnabled:            getEnvBool("PPROF_ENABLED", false),
		PprofUsername:           getEnv("PPROF_USERNAME", ""),
		PprofPassword:           getEnv("PPROF_PASSWORD", ""),
		MaxRequestBodySize:      0,
		DebugCacheHeaders:       getEnvBool("DEBUG_CACHE_HEADERS", false),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := api.Run(ctx, cfg); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return v
}

func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("Invalid integer value for %s: %v", key, err)
		}
		return i
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			log.Fatalf("Invalid boolean value for %s: %v", key, err)
		}
		return b
	}
	return defaultValue
}

func getEnvSlice(key, sep string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	return strings.Split(v, sep)
}
