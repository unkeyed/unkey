package cdc

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
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

// Client shares one Vitess connection between watches.
// Watches can run at the same time.
type Client struct {
	connection *grpc.ClientConn
	client     vtgateservice.VitessClient
	keyspace   string
	clock      clock.Clock
}

// ErrInvalidToken means the token cannot be read or does not match this watch.
// Discard it and start a new snapshot.
var ErrInvalidToken = errors.New("invalid CDC resume token")

// ErrExpired means the saved binlog position is no longer available.
// Start a new snapshot and remove destination records that no longer exist.
var ErrExpired = errors.New("CDC resume position expired")

// tablePattern prevents Vitess from treating a table name as a regex.
var tablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Rule names one table and the SELECT query used to filter its rows and columns.
// Do not build Query from unchecked user input.
type Rule struct {
	Table string `json:"table"`
	Query string `json:"query"`
}

// Event contains either a FIELD/ROW change or a checkpoint token, never both.
// Apply Change or save ResumeToken before returning from the callback.
type Event struct {
	Change      *binlog.VEvent
	ResumeToken []byte
}

// Config selects the Vitess endpoint and keyspace. TLS is on by default.
// Insecure is for local development. New requires both Username and Password
// or neither, and rejects credentials without TLS.
type Config struct {
	Address  string `toml:"address"`
	Keyspace string `toml:"keyspace"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Insecure bool   `toml:"insecure"`
}

// New checks settings but does not connect. [Client.Watch] opens the connection.
// It returns nil on error. Close the client when it is no longer needed.
func New(cfg Config) (*Client, error) {
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
	return &Client{connection: connection, client: vtgateservice.NewVitessClient(connection), keyspace: cfg.Keyspace, clock: clock.New()}, nil
}

// Close closes the connection used by all watches. It does not wait for callbacks.
// Cancel the watches and wait for them to stop before calling Close.
func (c *Client) Close() error { return c.connection.Close() }

// Watch copies current rows when token is empty, then follows changes.
// It calls apply for each change or checkpoint, one at a time. The callback
// must be non-nil, must not edit events, and must finish applying each event
// before returning. Separate watches can call it at the same time.
//
// A checkpoint marks a safe place to resume. Callback errors stop the watch
// without checkpointing that transaction. Callers must handle repeated events,
// save tokens, and reconnect. Rules must not change during the call.
// Tokens match the exact rules, including their order.
//
// Transactions with matching rows checkpoint at once. After the first checkpoint,
// other progress checkpoints at most once per 30 seconds. Watch times out after
// 30 seconds without a response. Time spent in callbacks does not count.
// It returns callback, connection, or context errors when it stops.
// [ErrInvalidToken] and [ErrExpired] require a new snapshot.
func (c *Client) Watch(ctx context.Context, rules []Rule, token []byte, apply func(Event) error) error {
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
			if len(token) > 0 && status.Code(err) == codes.Unknown &&
				(strings.Contains(err.Error(), "(errno 1236)") || strings.Contains(err.Error(), "(errno 1789)")) {
				return fmt.Errorf("%w: %w", ErrExpired, err)
			}
			return err
		}
		for _, event := range response.Events {
			if event.Type == binlog.VEventType_FIELD || event.Type == binlog.VEventType_ROW {
				if err := apply(Event{Change: event, ResumeToken: nil}); err != nil {
					return err
				}
			}
			if err := checkpoints.advance(event, rules, c.clock, apply); err != nil {
				return err
			}
		}
	}
}

// vstreamFilter checks table names and queries before starting a watch.
func vstreamFilter(rules []Rule) (*binlog.Filter, error) {
	if len(rules) == 0 {
		return nil, errors.New("CDC requires at least one table rule")
	}
	filter := &binlog.Filter{Rules: nil} //nolint:exhaustruct // No replication workflow metadata is needed.
	for _, rule := range rules {
		if !tablePattern.MatchString(rule.Table) || rule.Query == "" {
			return nil, errors.New("CDC requires literal table names and nonempty queries")
		}
		filter.Rules = append(filter.Rules, &binlog.Rule{Match: rule.Table, Filter: rule.Query}) //nolint:exhaustruct // No replication workflow metadata is needed.
	}
	return filter, nil
}

// checkpointState separates unfinished transactions from safe resume positions.
type checkpointState struct {
	pending        *binlog.VGtid
	committed      *binlog.VGtid
	lastCheckpoint time.Time
	changed        bool
}

// advance updates progress after the caller has delivered the event.
// A heartbeat can send a checkpoint, but only for a finished transaction.
func (s *checkpointState) advance(event *binlog.VEvent, rules []Rule, clock clock.Clock, apply func(Event) error) error {
	if event.Type == binlog.VEventType_FIELD || event.Type == binlog.VEventType_ROW {
		s.changed = s.changed || len(event.GetRowEvent().GetRowChanges()) > 0
	}
	if event.Type == binlog.VEventType_VGTID {
		s.pending = event.Vgtid
	}
	boundary := event.Type == binlog.VEventType_COMMIT || event.Type == binlog.VEventType_DDL || event.Type == binlog.VEventType_OTHER
	flush := false
	if boundary && s.pending != nil {
		s.committed = s.pending
		s.pending = nil
		flush = s.changed
		s.changed = false
	}
	if s.committed == nil || (!flush && clock.Now().Sub(s.lastCheckpoint) < 30*time.Second) {
		return nil
	}
	encoded, err := protojson.Marshal(s.committed)
	if err != nil {
		return err
	}
	next, err := json.Marshal(resumeToken{Rules: rules, Position: encoded})
	if err != nil {
		return err
	}
	if err := apply(Event{Change: nil, ResumeToken: next}); err != nil {
		return err
	}
	s.committed = nil
	s.lastCheckpoint = clock.Now()
	return nil
}

// streamResult carries a response or error from the read goroutine.
type streamResult struct {
	response *vtgate.VStreamResponse
	err      error
}

// receive limits the wait for a response, not the time spent in callbacks.
// If the context is canceled, its error takes priority over a read error.
func (c *Client) receive(ctx context.Context, responses <-chan streamResult) (*vtgate.VStreamResponse, error) {
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

// readVStream lets Watch time out a blocked Recv call.
// It reads at most one response ahead and stops when the context is canceled.
func readVStream(ctx context.Context, stream vtgateservice.Vitess_VStreamClient) <-chan streamResult {
	responses := make(chan streamResult)
	go func() {
		for {
			response, err := stream.Recv()
			select {
			case responses <- streamResult{response: response, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return responses
}

// resumeToken stores a stream position with its filter rules.
// This prevents resuming with different rules. It does not grant access.
type resumeToken struct {
	Rules    []Rule          `json:"rules"`
	Position json.RawMessage `json:"position"`
}

// position starts a snapshot for an empty token or checks a saved position.
// Invalid tokens return nil and an error wrapping [ErrInvalidToken].
func (c *Client) position(rules []Rule, token []byte) (*binlog.VGtid, error) {
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
