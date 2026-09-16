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

// Client shares a Vitess connection across independent, concurrent watches.
// Callers own reconnects and checkpoint storage; the client retains neither.
type Client struct {
	connection *grpc.ClientConn
	client     vtgateservice.VitessClient
	keyspace   string
	clock      clock.Clock
}

// ErrInvalidToken identifies malformed tokens or tokens for another filter,
// keyspace, or snapshot table. Callers must discard them before reconnecting.
var ErrInvalidToken = errors.New("invalid CDC resume token")

// ErrExpired identifies unavailable binlogs when resuming a saved position.
// Recovery requires a fresh snapshot and destination reconciliation.
var ErrExpired = errors.New("CDC resume position expired")

// tablePattern prevents table rules from being interpreted as Vitess regex rules.
var tablePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Rule selects a literal table and a Vitess-supported SELECT projection/filter.
// Query is trusted application configuration, not unvalidated user input.
type Rule struct {
	Table string `json:"table"`
	Query string `json:"query"`
}

// Config selects the Vitess endpoint and keyspace. TLS uses system roots unless
// Insecure is set for local development. New rejects credentials without TLS
// and requires Username and Password to be supplied together.
type Config struct {
	Address  string `toml:"address"`
	Keyspace string `toml:"keyspace"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Insecure bool   `toml:"insecure"`
}

// New validates connection settings and creates a client without connecting.
// Endpoint availability and authentication are checked by [Client.Watch].
// It returns nil on error; callers must close a successfully created client.
func New(cfg Config) (*Client, error) {
	if cfg.Address == "" || cfg.Keyspace == "" {
		return nil, errors.New("vstream.address and vstream.keyspace are required")
	}
	if (cfg.Username == "") != (cfg.Password == "") {
		return nil, errors.New("vstream username and password must be set together")
	}
	transport := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12}) //nolint:exhaustruct // Use system roots and secure TLS defaults.
	if cfg.Insecure {
		if cfg.Username != "" {
			return nil, errors.New("VStream credentials require TLS")
		}
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

// Close closes the transport shared by all watches. It does not wait for their
// callbacks; callers should cancel and wait for watches before closing.
func (c *Client) Close() error { return c.connection.Close() }

// Watch copies current rows for an empty token, then follows live changes.
// It delivers FIELD and ROW events synchronously, preserving table and shard
// identity, types, NULLs, and before/after images. Callback errors stop the watch
// without checkpointing that transaction. Rules must remain unchanged during
// the call; tokens bind to their exact contents and order. Callers own decoding,
// idempotent application, token persistence, and reconnects.
//
// Both callbacks must be non-nil and must treat events as read-only. Callbacks
// are serial within a watch, but concurrent watches can invoke them concurrently.
// Relevant commits checkpoint promptly. After the first checkpoint, unrelated
// checkpoints are sent at most once per 30 seconds. Upstream waits time out after
// 30 seconds; callback execution does not count toward this timeout.
// Watch returns callback, transport, or context errors when it stops. Resume
// failures matching [ErrInvalidToken] or [ErrExpired] require a new snapshot.
func (c *Client) Watch(ctx context.Context, rules []Rule, token []byte, change func(*binlog.VEvent) error, checkpoint func([]byte) error) error {
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
				if err := change(event); err != nil {
					return err
				}
			}
			if err := checkpoints.advance(event, rules, c.clock, checkpoint); err != nil {
				return err
			}
		}
	}
}

// vstreamFilter validates explicit table rules before opening a subscription.
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

// checkpointState keeps uncommitted positions separate from replay-safe progress.
// Its zero value starts a watch with no acknowledged transaction.
type checkpointState struct {
	pending        *binlog.VGtid
	committed      *binlog.VGtid
	lastCheckpoint time.Time
	changed        bool
}

// advance observes events only after their delivery succeeds. It publishes
// relevant commits immediately and coalesces unrelated committed positions.
// Heartbeats can flush a committed position, never the pending transaction.
func (s *checkpointState) advance(event *binlog.VEvent, rules []Rule, clock clock.Clock, checkpoint func([]byte) error) error {
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
	if err := checkpoint(next); err != nil {
		return err
	}
	s.committed = nil
	s.lastCheckpoint = clock.Now()
	return nil
}

// streamResult carries one receive outcome across the upstream timeout boundary.
type streamResult struct {
	response *vtgate.VStreamResponse
	err      error
}

// receive limits only upstream waiting, so slow callbacks cannot expire a watch.
// Context cancellation takes precedence over simultaneous receive failures.
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

// readVStream isolates blocking Recv calls so Watch can time out upstream waits
// without timing out callbacks. Its unbuffered channel bounds read-ahead to one
// response; cancellation releases a blocked receive or send.
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

// resumeToken binds a complete cursor to its subscription to prevent accidental
// cross-filter resumes. It is not an authorization credential.
type resumeToken struct {
	Rules    []Rule          `json:"rules"`
	Position json.RawMessage `json:"position"`
}

// position selects a new snapshot for an empty token or validates a saved cursor.
// Failed validation returns nil and wraps ErrInvalidToken, never a partial cursor.
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

// basicAuth holds a pre-encoded username/password pair for TLS-only VStream RPCs.
type basicAuth string

// GetRequestMetadata supplies PlanetScale-compatible HTTP Basic authentication.
func (a basicAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Basic " + string(a)}, nil
}

// RequireTransportSecurity prevents gRPC from sending credentials over plaintext.
func (basicAuth) RequireTransportSecurity() bool { return true }
