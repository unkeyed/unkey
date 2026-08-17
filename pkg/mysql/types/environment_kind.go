package dbtype

import (
	"database/sql/driver"
	"fmt"
)

// EnvironmentKind classifies how deployments in an environment behave.
type EnvironmentKind string

const (
	EnvironmentKindProduction EnvironmentKind = "production"
	EnvironmentKindPreview    EnvironmentKind = "preview"
)

// IsProduction reports whether deployments serve production traffic.
func (k EnvironmentKind) IsProduction() bool {
	return k == EnvironmentKindProduction
}

// IsPreview reports whether deployments have preview lifecycle behavior.
func (k EnvironmentKind) IsPreview() bool {
	return k == EnvironmentKindPreview
}

// Scan implements the sql.Scanner interface.
func (k *EnvironmentKind) Scan(src any) error {
	switch value := src.(type) {
	case []byte:
		*k = EnvironmentKind(value)
	case string:
		*k = EnvironmentKind(value)
	default:
		return fmt.Errorf("unsupported scan type for EnvironmentKind: %T", src)
	}
	return nil
}

// Value implements the driver.Valuer interface.
func (k EnvironmentKind) Value() (driver.Value, error) {
	return string(k), nil
}
