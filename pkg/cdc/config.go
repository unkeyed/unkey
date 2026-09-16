package cdc

// Config fixes a client's rules on a shared connection.
type Config struct {
	Connection *Connection
	Rules      []Rule
}
