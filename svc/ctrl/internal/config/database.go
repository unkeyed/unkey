package config

import "fmt"

// DatabaseConfig configures the single read-write MySQL connection used by
// control plane processes.
type DatabaseConfig struct {
	// Primary is the MySQL DSN used for all control plane reads and writes.
	Primary string `toml:"primary" config:"required,nonempty"`

	// Deprecated: control plane processes don't use a read-only replica.
	ReadonlyReplica string `toml:"readonly_replica"`
}

// Validate rejects read-only replica configuration.
func (c DatabaseConfig) Validate() error {
	if c.ReadonlyReplica != "" {
		return fmt.Errorf("database.readonly_replica is not supported for svc/ctrl; use database.primary")
	}
	return nil
}
