package cdc

// ConnectionConfig selects the Vitess endpoint and keyspace. TLS is on by default.
// Insecure is for local development. Credentials require TLS and both fields.
type ConnectionConfig struct {
	Address  string `toml:"address"`
	Keyspace string `toml:"keyspace"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Insecure bool   `toml:"insecure"`
}
