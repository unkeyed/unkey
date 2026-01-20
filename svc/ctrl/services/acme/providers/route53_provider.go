package providers

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type Route53Config struct {
	DB              db.Database
	Logger          logging.Logger
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	DomainCache     cache.Cache[string, db.CustomDomain]
	// HostedZoneID bypasses zone auto-discovery. Required when domains have CNAMEs
	// that would confuse the zone lookup (e.g., wildcard CNAMEs to load balancers).
	HostedZoneID string
}

// validateAWSCredentials checks AWS credentials for common issues and returns sanitized values.
// Returns an error if credentials are invalid or have issues that will cause SignatureDoesNotMatch errors.
func validateAWSCredentials(logger logging.Logger, accessKeyID, secretAccessKey string) (string, string, error) {
	// Trim whitespace (including newlines) - common issue with Kubernetes secrets
	cleanKeyID := strings.TrimSpace(accessKeyID)
	cleanSecret := strings.TrimSpace(secretAccessKey)

	// Check if trimming removed anything (indicates potential issue)
	if cleanKeyID != accessKeyID {
		logger.Warn("AWS access key ID had leading/trailing whitespace - this was trimmed",
			"original_length", len(accessKeyID),
			"trimmed_length", len(cleanKeyID),
		)
	}
	if cleanSecret != secretAccessKey {
		logger.Warn("AWS secret access key had leading/trailing whitespace - this was trimmed. "+
			"This is a common cause of SignatureDoesNotMatch errors!",
			"original_length", len(secretAccessKey),
			"trimmed_length", len(cleanSecret),
			"had_newline", strings.Contains(secretAccessKey, "\n"),
		)
	}

	// Validate access key ID format (starts with AKIA, ASIA, or AROA and is 20 chars)
	if cleanKeyID == "" {
		return "", "", fmt.Errorf("AWS access key ID is empty")
	}
	if len(cleanKeyID) != 20 {
		logger.Warn("AWS access key ID has unexpected length",
			"expected", 20,
			"actual", len(cleanKeyID),
		)
	}
	accessKeyPattern := regexp.MustCompile(`^(AKIA|ASIA|AIDA|AROA|ANPA|ANVA|AGPA)[A-Z0-9]{16}$`)
	if !accessKeyPattern.MatchString(cleanKeyID) {
		logger.Warn("AWS access key ID does not match expected format (should start with AKIA/ASIA/etc and be 20 alphanumeric chars)",
			"key_id_prefix", cleanKeyID[:min(4, len(cleanKeyID))],
		)
	}

	// Validate secret access key
	if cleanSecret == "" {
		return "", "", fmt.Errorf("AWS secret access key is empty")
	}
	if len(cleanSecret) != 40 {
		logger.Warn("AWS secret access key has unexpected length (expected 40 chars)",
			"actual_length", len(cleanSecret),
		)
	}

	// Check for common encoding issues
	if strings.Contains(cleanSecret, "%") {
		logger.Warn("AWS secret access key contains '%' - this may indicate URL encoding issues")
	}
	if strings.Contains(cleanSecret, "\\") {
		logger.Warn("AWS secret access key contains backslash - this may indicate escape sequence issues")
	}

	return cleanKeyID, cleanSecret, nil
}

// NewRoute53Provider creates a new DNS-01 challenge provider using AWS Route53.
//
// Important: LEGO_DISABLE_CNAME_SUPPORT must be set to "true" before calling this
// function to prevent lego from following wildcard CNAMEs and failing zone lookup.
// This should be done once at application startup (see run.go).
//
// HostedZoneID is optional - if empty, lego will auto-discover the hosted zone
// based on the domain. Provide it only if auto-discovery fails (e.g., due to CNAMEs).
func NewRoute53Provider(cfg Route53Config) (*Provider, error) {
	// Validate and sanitize credentials before use
	accessKeyID, secretAccessKey, err := validateAWSCredentials(cfg.Logger, cfg.AccessKeyID, cfg.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("invalid AWS credentials: %w", err)
	}

	config := route53.NewDefaultConfig()
	config.PropagationTimeout = time.Minute * 5
	config.TTL = 60 * 10 // 10 minutes
	config.AccessKeyID = accessKeyID
	config.SecretAccessKey = secretAccessKey
	config.Region = cfg.Region
	config.WaitForRecordSetsChanged = true

	// Only set HostedZoneID if provided - otherwise let lego auto-discover
	if cfg.HostedZoneID != "" {
		config.HostedZoneID = cfg.HostedZoneID
	}

	cfg.Logger.Info("Route53 provider configured",
		"region", cfg.Region,
		"hosted_zone_id", cfg.HostedZoneID,
		"access_key_prefix", accessKeyID[:min(8, len(accessKeyID))],
	)

	dns, err := route53.NewDNSProviderConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Route53 DNS provider: %w", err)
	}

	return NewProvider(ProviderConfig{
		DB:          cfg.DB,
		Logger:      cfg.Logger,
		DNS:         dns,
		DomainCache: cfg.DomainCache,
	})
}
