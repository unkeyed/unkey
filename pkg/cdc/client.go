package cdc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
	"vitess.io/vitess/go/vt/proto/topodata"
	"vitess.io/vitess/go/vt/proto/vtgate"
	"vitess.io/vitess/go/vt/proto/vtgateservice"
)

// Client watches one set of rules and remembers its last safe resume token.
// Use separate clients for separate consumers. Methods must not run concurrently.
type Client struct {
	connection *Connection
	rules      []Rule
	token      []byte
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
// Forward delivers these events so relays can pass checkpoints to their consumers.
type Event struct {
	Change      *binlog.VEvent
	ResumeToken []byte
}

// Config fixes a client's rules on a shared connection.
type Config struct {
	Connection *Connection
	Rules      []Rule
}

// New checks and copies the rules without opening a stream.
// The first Watch starts a snapshot. Later calls reuse the client's progress.
// It returns nil on error. The caller owns Connection, which must not be nil.
func New(cfg Config) (*Client, error) {
	if err := assert.NotNil(cfg.Connection, "CDC connection is required"); err != nil {
		return nil, err
	}
	if _, err := vstreamFilter(cfg.Rules); err != nil {
		return nil, err
	}
	return &Client{connection: cfg.Connection, rules: slices.Clone(cfg.Rules), token: nil}, nil
}

// Watch calls apply for FIELD and ROW changes, one at a time, and saves checkpoints
// internally. The callback must be non-nil, must not edit events, and must finish
// applying each change before returning. Repeated changes must be safe to apply.
// Errors stop the watch. Calling Watch again resumes from the last safe token.
// It does not retry automatically. After [ErrExpired], the next call starts a
// snapshot. The caller must remove destination records that no longer exist.
func (c *Client) Watch(ctx context.Context, apply func(*binlog.VEvent) error) error {
	err := c.connection.Forward(ctx, c.rules, c.token, func(event Event) error {
		if event.Change != nil {
			return apply(event.Change)
		}
		c.token = event.ResumeToken
		return nil
	})
	if errors.Is(err, ErrExpired) {
		c.token = nil
	}
	return err
}

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
