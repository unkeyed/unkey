package cdc

import "github.com/unkeyed/unkey/pkg/assert"

// Config selects a client's Vitess endpoint, keyspace, and rules. TLS is on by default.
// Insecure is for local development. Credentials require TLS and both fields.
type Config struct {
	Address  string `toml:"address"`
	Keyspace string `toml:"keyspace"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Insecure bool   `toml:"insecure"`
	Rules    []Rule `toml:"-"`
}

// ValidateEndpoint checks the endpoint and credentials before any watches start.
// New also checks Rules, which relays can set for each watch.
func (c Config) ValidateEndpoint() error {
	return assert.All(
		assert.NotEmpty(c.Address, "vstream.address is required"),
		assert.NotEmpty(c.Keyspace, "vstream.keyspace is required"),
		assert.True((c.Username == "") == (c.Password == ""), "vstream username and password must be set together"),
		assert.True(!c.Insecure || c.Username == "", "VStream credentials require TLS"),
	)
}
