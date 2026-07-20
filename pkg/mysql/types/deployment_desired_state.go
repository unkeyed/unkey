package dbtype

import (
	"database/sql/driver"
	"fmt"
)

// DeploymentsDesiredState is the canonical deployment intent enum, shared by every
// db package instead of each sqlc config regenerating its own copy. See
// DeploymentsStatus for how the generated packages point at it via a go_type override.
type DeploymentsDesiredState string

const (
	DeploymentsDesiredStateRunning DeploymentsDesiredState = "running"
	DeploymentsDesiredStateStopped DeploymentsDesiredState = "stopped"
)

func (e *DeploymentsDesiredState) Scan(src any) error {
	switch s := src.(type) {
	case []byte:
		*e = DeploymentsDesiredState(s)
	case string:
		*e = DeploymentsDesiredState(s)
	default:
		return fmt.Errorf("unsupported scan type for DeploymentsDesiredState: %T", src)
	}
	return nil
}

type NullDeploymentsDesiredState struct {
	DeploymentsDesiredState DeploymentsDesiredState
	Valid                   bool // Valid is true if DeploymentsDesiredState is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullDeploymentsDesiredState) Scan(value any) error {
	if value == nil {
		ns.DeploymentsDesiredState, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.DeploymentsDesiredState.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullDeploymentsDesiredState) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.DeploymentsDesiredState), nil
}
