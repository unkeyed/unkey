package mcp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Config is read from environment variables because Unkey Deploy configures
// workloads that way and injects PORT.
type Config struct {
	// Port is the HTTP listen port, from PORT. Defaults to 8080.
	Port int
	// PublicURL is the origin clients connect to, from PUBLIC_URL, such as
	// https://mcp.unkey.com.
	PublicURL string
	// APIURL receives forwarded tool calls, from API_URL, such as
	// https://api.unkey.com.
	APIURL string
	// OAuthIssuer is the authorization server named in protected resource
	// metadata, from OAUTH_ISSUER. Defaults to APIURL. It must equal the API's
	// OAuth issuer.
	OAuthIssuer string
	// ClickHouseURL receives one analytics row per tool call, from
	// CLICKHOUSE_URL. Empty disables analytics.
	ClickHouseURL string
}

// ConfigFromEnv reads [Config] through getenv, normally [os.Getenv].
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := Config{
		Port:          8080,
		PublicURL:     strings.TrimSuffix(getenv("PUBLIC_URL"), "/"),
		APIURL:        strings.TrimSuffix(getenv("API_URL"), "/"),
		OAuthIssuer:   getenv("OAUTH_ISSUER"),
		ClickHouseURL: getenv("CLICKHOUSE_URL"),
	}
	if raw := getenv("PORT"); raw != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("PORT %q is not a TCP port", raw)
		}
		cfg.Port = port
	}
	if cfg.PublicURL == "" || cfg.APIURL == "" {
		return Config{}, errors.New("PUBLIC_URL and API_URL are required")
	}
	if cfg.OAuthIssuer == "" {
		cfg.OAuthIssuer = cfg.APIURL
	}
	return cfg, nil
}
