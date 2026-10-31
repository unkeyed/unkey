package dbtype

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// DeviceAuthorizationOAuth is the OAuth 2.1 authorization request a device
// authorization answers. It is stored in the nullable `oauth` JSON column and
// read through a pointer: nil means the row is a CLI device login.
type DeviceAuthorizationOAuth struct {
	ClientID      string `json:"client_id"`
	RedirectURI   string `json:"redirect_uri"`
	CodeChallenge string `json:"code_challenge"`
	State         string `json:"state"`
	Resource      string `json:"resource"`
}

// Scan implements sql.Scanner for reading the JSON object from the database.
func (o *DeviceAuthorizationOAuth) Scan(value any) error {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("DeviceAuthorizationOAuth.Scan: expected []byte or string, got %T", value)
	}
	return json.Unmarshal(bytes, o)
}

// Value implements driver.Valuer. It returns a string so MySQL accepts it for
// the JSON column when the driver interpolates parameters.
func (o DeviceAuthorizationOAuth) Value() (driver.Value, error) {
	bytes, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}
