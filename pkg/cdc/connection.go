package cdc

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
	"vitess.io/vitess/go/vt/proto/topodata"
	"vitess.io/vitess/go/vt/proto/vtgate"
	"vitess.io/vitess/go/vt/proto/vtgateservice"
)

// Connection shares one Vitess connection between independently configured clients.
type Connection struct {
	connection *grpc.ClientConn
	client     vtgateservice.VitessClient
	keyspace   string
	clock      clock.Clock
}

// NewConnection checks settings but does not connect until a stream starts.
// It returns nil on error. Close the connection when all streams have stopped.
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
	return &Connection{connection: connection, client: vtgateservice.NewVitessClient(connection), keyspace: cfg.Keyspace, clock: clock.New()}, nil
}

// Close closes the connection used by all clients. It does not wait for callbacks.
// Cancel the watches and wait for them to stop before calling Close.
func (c *Connection) Close() error { return c.connection.Close() }

// Forward is for relays that must send checkpoints to another consumer.
// It sends changes and checkpoints in order without saving progress. Reconnect
// with the downstream consumer's token. An empty token starts a snapshot.
// Calls can run concurrently. Rules and tokens must not change during a call.
// The callback must be non-nil and must not edit events. Errors stop the stream;
// [ErrInvalidToken] and [ErrExpired] require a snapshot with an empty token.
//
// Matching transactions checkpoint at once. Other progress checkpoints at most
// once per 30 seconds after the first checkpoint. The upstream wait times out
// after 30 seconds; time spent in callbacks does not count.
func (c *Connection) Forward(ctx context.Context, rules []Rule, token []byte, send func(Event) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	filter, err := vstreamFilter(rules)
	if err != nil {
		return err
	}
	position, err := c.position(rules, token)
	if err != nil {
		return err
	}
	stream, err := c.client.VStream(ctx, &vtgate.VStreamRequest{ //nolint:exhaustruct // Leave unrelated protobuf options at their defaults.
		TabletType: topodata.TabletType_PRIMARY,
		Vgtid:      position,
		Filter:     filter,
		Flags:      &vtgate.VStreamFlags{HeartbeatInterval: 5}, //nolint:exhaustruct // Do not enable transaction chunking or optional stream features.
	})
	if err != nil {
		return err
	}
	responses := readVStream(ctx, stream)
	var checkpoints checkpointState
	for {
		response, err := c.receive(ctx, responses)
		if err != nil {
			if status.Code(err) == codes.Unknown &&
				(strings.Contains(err.Error(), "(errno 1236)") || strings.Contains(err.Error(), "(errno 1789)")) {
				return fmt.Errorf("%w: %w", ErrExpired, err)
			}
			return err
		}
		for _, event := range response.Events {
			if event.Type == binlog.VEventType_FIELD || event.Type == binlog.VEventType_ROW {
				if err := send(Event{Change: event, ResumeToken: nil}); err != nil {
					return err
				}
			}
			if err := checkpoints.advance(event, rules, c.clock, send); err != nil {
				return err
			}
		}
	}
}

// receive limits the wait for a response, not the time spent in callbacks.
// If the context is canceled, its error takes priority over a read error.
func (c *Connection) receive(ctx context.Context, responses <-chan streamResult) (*vtgate.VStreamResponse, error) {
	waiting := c.clock.NewTicker(30 * time.Second)
	defer waiting.Stop()
	var result streamResult
	select {
	case result = <-responses:
	case <-waiting.C():
		result.err = status.Error(codes.Unavailable, "VStream stalled for 30 seconds")
	case <-ctx.Done():
		result.err = ctx.Err()
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return result.response, result.err
}

// position starts a snapshot for an empty token or checks a saved position.
// Invalid tokens return nil and an error wrapping [ErrInvalidToken].
func (c *Connection) position(rules []Rule, token []byte) (*binlog.VGtid, error) {
	if len(token) == 0 {
		return &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: c.keyspace}}}, nil
	}
	var saved resumeToken
	if err := json.Unmarshal(token, &saved); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if !slices.Equal(saved.Rules, rules) {
		return nil, fmt.Errorf("%w: filter mismatch", ErrInvalidToken)
	}
	position := &binlog.VGtid{ShardGtids: nil}
	if err := protojson.Unmarshal(saved.Position, position); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if len(position.ShardGtids) == 0 {
		return nil, fmt.Errorf("%w: missing shard positions", ErrInvalidToken)
	}
	for _, shard := range position.ShardGtids {
		if shard.Keyspace != c.keyspace || shard.Gtid == "" {
			return nil, fmt.Errorf("%w: invalid shard position", ErrInvalidToken)
		}
		for _, table := range shard.TablePKs {
			if !slices.ContainsFunc(rules, func(rule Rule) bool { return rule.Table == table.GetTableName() }) {
				return nil, fmt.Errorf("%w: invalid snapshot table", ErrInvalidToken)
			}
		}
	}
	return position, nil
}

// basicAuth holds the base64-encoded username and password. It requires TLS.
type basicAuth string

// GetRequestMetadata sends the credentials in PlanetScale's HTTP Basic format.
func (a basicAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Basic " + string(a)}, nil
}

// RequireTransportSecurity prevents sending credentials without TLS.
func (basicAuth) RequireTransportSecurity() bool { return true }
