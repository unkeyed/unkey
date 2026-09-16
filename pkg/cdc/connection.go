package cdc

import (
	"context"
	"crypto/tls"
	"encoding/base64"

	"github.com/unkeyed/unkey/pkg/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"vitess.io/vitess/go/vt/proto/vtgateservice"
)

// Connection shares one Vitess connection between independently configured clients.
type Connection struct {
	connection *grpc.ClientConn
	client     vtgateservice.VitessClient
	keyspace   string
}

// ConnectionConfig selects the Vitess endpoint and keyspace. TLS is on by default.
// Insecure is for local development. Credentials require TLS and both fields.
type ConnectionConfig struct {
	Address  string `toml:"address"`
	Keyspace string `toml:"keyspace"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Insecure bool   `toml:"insecure"`
}

// NewConnection checks settings but does not connect until a client starts watching.
// It returns nil on error. Close the connection when all clients have stopped.
func NewConnection(cfg ConnectionConfig) (*Connection, error) {
	if err := assert.All(
		assert.NotEmpty(cfg.Address, "vstream.address is required"),
		assert.NotEmpty(cfg.Keyspace, "vstream.keyspace is required"),
		assert.True((cfg.Username == "") == (cfg.Password == ""), "vstream username and password must be set together"),
		assert.True(!cfg.Insecure || cfg.Username == "", "VStream credentials require TLS"),
	); err != nil {
		return nil, err
	}
	transport := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}) //nolint:exhaustruct // Use system roots and secure TLS defaults.
	if cfg.Insecure {
		transport = insecure.NewCredentials()
	}
	options := []grpc.DialOption{grpc.WithTransportCredentials(transport)}
	if cfg.Username != "" {
		options = append(options, grpc.WithPerRPCCredentials(basicAuth(base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)))))
	}
	connection, err := grpc.NewClient(cfg.Address, options...)
	if err != nil {
		return nil, err
	}
	return &Connection{connection: connection, client: vtgateservice.NewVitessClient(connection), keyspace: cfg.Keyspace}, nil
}

// Close closes the connection used by all clients. It does not wait for callbacks.
// Cancel the watches and wait for them to stop before calling Close.
func (c *Connection) Close() error { return c.connection.Close() }

// basicAuth holds the base64-encoded username and password. It requires TLS.
type basicAuth string

// GetRequestMetadata sends the credentials in PlanetScale's HTTP Basic format.
func (a basicAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Basic " + string(a)}, nil
}

// RequireTransportSecurity prevents sending credentials without TLS.
func (basicAuth) RequireTransportSecurity() bool { return true }
