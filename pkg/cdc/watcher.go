package cdc

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
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

// Watcher copies matching rows and follows their changes through Vitess.
// It is safe for concurrent use. Watch rejects overlapping calls.
type Watcher struct {
	connection *grpc.ClientConn
	client     vtgateservice.VitessClient
	keyspace   string
	clock      clock.Clock
	rules      []Rule
	watching   atomic.Bool
}

// New checks settings and copies rules without opening a stream.
// It returns nil on error. Close the watcher when no more watches are needed.
func New(cfg Config) (*Watcher, error) {
	if err := cfg.ValidateEndpoint(); err != nil {
		return nil, err
	}
	if _, err := vstreamFilter(cfg.Rules); err != nil {
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
	return &Watcher{
		connection: connection,
		client:     vtgateservice.NewVitessClient(connection),
		keyspace:   cfg.Keyspace,
		clock:      clock.New(),
		rules:      slices.Clone(cfg.Rules),
		watching:   atomic.Bool{},
	}, nil
}

// Close releases the connection and interrupts the active stream.
// It does not wait for a callback already in progress.
func (c *Watcher) Close() error { return c.connection.Close() }

// Watch sends changes and checkpoints in order without saving progress.
// Reconnect with the consumer's last applied token. An empty token starts a
// snapshot. The token must not change during a call. Changes can repeat.
// The callback must be non-nil and must not edit events. Errors stop the stream;
// [ErrInvalidToken] and [ErrExpired] require a snapshot with an empty token.
//
// Matching transactions checkpoint at once. Other progress checkpoints at most
// once per 30 seconds after the first checkpoint. The upstream wait times out
// after 30 seconds; time spent in callbacks does not count.
func (c *Watcher) Watch(ctx context.Context, token []byte, send func(Event) error) error {
	if err := assert.True(c.watching.CompareAndSwap(false, true), "CDC watcher already has an active watch"); err != nil {
		return err
	}
	defer c.watching.Store(false)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	filter, err := vstreamFilter(c.rules)
	if err != nil {
		return err
	}
	position, err := c.position(token)
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
			if err := checkpoints.advance(event, c.rules, c.clock, send); err != nil {
				return err
			}
		}
	}
}

// receive limits the wait for a response, not the time spent in callbacks.
// If the context is canceled, its error takes priority over a read error.
func (c *Watcher) receive(ctx context.Context, responses <-chan streamResult) (*vtgate.VStreamResponse, error) {
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
func (c *Watcher) position(token []byte) (*binlog.VGtid, error) {
	if len(token) == 0 {
		return &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: c.keyspace}}}, nil
	}
	var saved resumeToken
	if err := json.Unmarshal(token, &saved); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if !slices.Equal(saved.Rules, c.rules) {
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
			if !slices.ContainsFunc(c.rules, func(rule Rule) bool { return rule.Table == table.GetTableName() }) {
				return nil, fmt.Errorf("%w: invalid snapshot table", ErrInvalidToken)
			}
		}
	}
	return position, nil
}
