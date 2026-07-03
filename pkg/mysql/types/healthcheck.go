package dbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Healthcheck configures HTTP health probes for a deployment.
// A nil *Healthcheck means no healthcheck is configured.
type Healthcheck struct {
	Method              string `json:"method"`              // "GET" or "POST"
	Path                string `json:"path"`                // e.g. "/healthz"
	IntervalSeconds     int    `json:"intervalSeconds"`     // how often to probe
	TimeoutSeconds      int    `json:"timeoutSeconds"`      // per-probe timeout
	FailureThreshold    int    `json:"failureThreshold"`    // failures before restart
	InitialDelaySeconds int    `json:"initialDelaySeconds"` // wait before first probe
}

// NullHealthcheck wraps *Healthcheck for nullable JSON columns.
// Value is nil when the database column is NULL (no healthcheck configured).
type NullHealthcheck struct {
	Healthcheck *Healthcheck
	Valid       bool
}

// Scan implements sql.Scanner for reading JSON from the database.
func (h *NullHealthcheck) Scan(value interface{}) error {
	if value == nil {
		h.Healthcheck = nil
		h.Valid = false
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("NullHealthcheck.Scan: expected []byte or string, got %T", value)
	}

	if len(bytes) == 0 || string(bytes) == "null" {
		h.Healthcheck = nil
		h.Valid = false
		return nil
	}

	var hc Healthcheck
	if err := json.Unmarshal(bytes, &hc); err != nil {
		return err
	}

	h.Healthcheck = &hc
	h.Valid = true
	return nil
}

// Value implements driver.Valuer for writing JSON to the database.
func (h NullHealthcheck) Value() (driver.Value, error) {
	if !h.Valid || h.Healthcheck == nil {
		return nil, nil
	}

	bytes, err := json.Marshal(h.Healthcheck)
	if err != nil {
		return nil, err
	}

	return string(bytes), nil
}
