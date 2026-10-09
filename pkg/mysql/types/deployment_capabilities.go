package dbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// DeploymentCapabilities are decided once when a deployment is created.
// Running deployments keep them regardless of later flag or setting changes.
// They are stored as a JSON object in which a missing key is off.
type DeploymentCapabilities struct {
	PrivateNetworking bool `json:"private_networking"`
}

// Scan implements sql.Scanner for reading the JSON object from the database.
func (f *DeploymentCapabilities) Scan(value any) error {
	*f = DeploymentCapabilities{PrivateNetworking: false}

	var bytes []byte
	switch v := value.(type) {
	case nil:
		return nil
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("DeploymentCapabilities.Scan: expected []byte or string, got %T", value)
	}

	if len(bytes) == 0 {
		return nil
	}
	return json.Unmarshal(bytes, f)
}

// Value implements driver.Valuer for writing the JSON object to the database.
func (f DeploymentCapabilities) Value() (driver.Value, error) {
	bytes, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}
