package mcp

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const (
	// ResourceAPI is the path and resource name for the full API spike.
	ResourceAPI = "api"

	// ResourceCompute is the path and resource name for the Compute spike.
	ResourceCompute = "compute"
)

// Config is the spike's runtime configuration. Every value comes from the
// process environment. The spike has no embedded secrets.
type Config struct {
	// Issuer is the expected iss claim, https:// plus the AuthKit host.
	Issuer string

	// JWKSURL is the AuthKit JSON Web Key Set, issuer plus /oauth2/jwks.
	JWKSURL string

	// PublicBaseURL is the external origin clients use, with no path.
	PublicBaseURL string

	// ListenAddr is the plain HTTP address the process binds.
	ListenAddr string

	// LogClaims records decoded claim names and values after the signature
	// check. It never records the bearer token or its signature.
	LogClaims bool
}

// ConfigFromEnv reads AUTHKIT_DOMAIN, PUBLIC_BASE_URL, LISTEN_ADDR, and the
// optional MCP_SPIKE_LOG_CLAIMS flag.
func ConfigFromEnv(getenv func(string) string) (Config, error) {
	domain, err := authKitHost(getenv("AUTHKIT_DOMAIN"))
	if err != nil {
		return zeroConfig(), err
	}

	public := strings.TrimSpace(getenv("PUBLIC_BASE_URL"))
	if strings.HasSuffix(public, "/") {
		return zeroConfig(), errors.New("PUBLIC_BASE_URL must not have a trailing slash")
	}

	logClaims := false
	if raw := strings.TrimSpace(getenv("MCP_SPIKE_LOG_CLAIMS")); raw != "" {
		logClaims, err = strconv.ParseBool(raw)
		if err != nil {
			return zeroConfig(), errors.New("MCP_SPIKE_LOG_CLAIMS must be true or false")
		}
	}

	cfg := Config{
		Issuer:        "https://" + domain,
		JWKSURL:       "https://" + domain + "/oauth2/jwks",
		PublicBaseURL: public,
		ListenAddr:    strings.TrimSpace(getenv("LISTEN_ADDR")),
		LogClaims:     logClaims,
	}
	if err := cfg.validate(); err != nil {
		return zeroConfig(), err
	}
	return cfg, nil
}

// ResourceURL is the protected-resource identifier for one spike path.
func (c Config) ResourceURL(name string) string {
	return c.PublicBaseURL + "/" + name
}

// MetadataURL is the RFC 9728 metadata URL for one spike path.
func (c Config) MetadataURL(name string) string {
	return c.PublicBaseURL + "/.well-known/oauth-protected-resource/" + name
}

func (c Config) validate() error {
	if err := validateOrigin("issuer", c.Issuer, true); err != nil {
		return err
	}
	jwks, err := url.Parse(c.JWKSURL)
	if err != nil || jwks.Host == "" || (jwks.Scheme != "https" && jwks.Scheme != "http") || jwks.User != nil {
		return errors.New("JWKS URL must be an absolute http(s) URL")
	}
	if err := validateOrigin("PUBLIC_BASE_URL", c.PublicBaseURL, false); err != nil {
		return err
	}
	if c.ListenAddr == "" {
		return errors.New("LISTEN_ADDR is required")
	}
	return nil
}

func authKitHost(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("AUTHKIT_DOMAIN is required")
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" || parsed.Scheme != "https" {
			return "", errors.New("AUTHKIT_DOMAIN must be a host or an https origin")
		}
		if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return "", errors.New("AUTHKIT_DOMAIN must be a host or an https origin")
		}
		return parsed.Host, nil
	}
	if strings.Contains(raw, "/") {
		return "", errors.New("AUTHKIT_DOMAIN must be a host or an https origin")
	}
	return raw, nil
}

func validateOrigin(name string, raw string, httpsOnly bool) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.Scheme == "" {
		return fmt.Errorf("%s must be an absolute URL", name)
	}
	if httpsOnly && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use https", name)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("%s must use http or https", name)
	}
	if (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return fmt.Errorf("%s must be an origin with no path, query, or fragment", name)
	}
	if strings.HasSuffix(raw, "/") {
		return fmt.Errorf("%s must not have a trailing slash", name)
	}
	return nil
}

func zeroConfig() Config {
	return Config{
		Issuer:        "",
		JWKSURL:       "",
		PublicBaseURL: "",
		ListenAddr:    "",
		LogClaims:     false,
	}
}
