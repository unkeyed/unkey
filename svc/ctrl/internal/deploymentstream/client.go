package deploymentstream

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

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

var ErrInvalidToken = errors.New("invalid deployment resume token")
var ErrExpired = errors.New("deployment resume position expired")

type Config struct {
	Address  string `toml:"address"`
	Keyspace string `toml:"keyspace"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	Insecure bool   `toml:"insecure"`
}

type Client struct {
	connection *grpc.ClientConn
	client     vtgateservice.VitessClient
	keyspace   string
}

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
	return &Client{connection: connection, client: vtgateservice.NewVitessClient(connection), keyspace: cfg.Keyspace}, nil
}

func (c *Client) Close() error { return c.connection.Close() }

func (c *Client) Watch(ctx context.Context, region string, token []byte, change func(string) error, checkpoint func([]byte) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,48}$`).MatchString(region) {
		return fmt.Errorf("invalid region ID")
	}
	position, err := c.position(region, token)
	if err != nil {
		return err
	}
	stream, err := c.client.VStream(ctx, &vtgate.VStreamRequest{ //nolint:exhaustruct // Leave unrelated protobuf options at their defaults.
		TabletType: topodata.TabletType_PRIMARY,
		Vgtid:      position,
		Filter: &binlog.Filter{Rules: []*binlog.Rule{{ //nolint:exhaustruct // No replication workflow metadata is needed.
			Match:  "deployment_topology",
			Filter: fmt.Sprintf("select deployment_id from deployment_topology where region_id = '%s'", region),
		}}},
		Flags: &vtgate.VStreamFlags{HeartbeatInterval: 5}, //nolint:exhaustruct // Do not enable transaction chunking or optional stream features.
	})
	if err != nil {
		return err
	}
	var pending *binlog.VGtid
	for {
		response, err := stream.Recv()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			if len(token) > 0 && status.Code(err) == codes.Unknown &&
				(strings.Contains(err.Error(), "(errno 1236)") || strings.Contains(err.Error(), "(errno 1789)")) {
				return fmt.Errorf("%w: %w", ErrExpired, err)
			}
			return err
		}
		for _, event := range response.Events {
			if event.RowEvent != nil {
				for _, rowChange := range event.RowEvent.RowChanges {
					row := rowChange.After
					if row == nil {
						row = rowChange.Before
					}
					if row == nil || len(row.Lengths) != 1 || row.Lengths[0] <= 0 || row.Lengths[0] != int64(len(row.Values)) {
						return errors.New("invalid deployment ID in VStream row")
					}
					if err := change(string(row.Values)); err != nil {
						return err
					}
				}
			}
			if event.Type == binlog.VEventType_VGTID {
				pending = event.Vgtid
			}
			boundary := event.Type == binlog.VEventType_COMMIT || event.Type == binlog.VEventType_DDL || event.Type == binlog.VEventType_OTHER
			if boundary && pending != nil {
				encoded, err := protojson.Marshal(pending)
				if err != nil {
					return err
				}
				next, err := json.Marshal(resumeToken{Region: region, Position: encoded})
				if err != nil {
					return err
				}
				if err := checkpoint(next); err != nil {
					return err
				}
				pending = nil
			}
		}
	}
}

type resumeToken struct {
	Region   string          `json:"region"`
	Position json.RawMessage `json:"position"`
}

func (c *Client) position(region string, token []byte) (*binlog.VGtid, error) {
	if len(token) == 0 {
		return &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: c.keyspace}}}, nil
	}
	var saved resumeToken
	if err := json.Unmarshal(token, &saved); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if saved.Region != region {
		return nil, fmt.Errorf("%w: region mismatch", ErrInvalidToken)
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
	}
	return position, nil
}

type basicAuth string

func (a basicAuth) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Basic " + string(a)}, nil
}

func (basicAuth) RequireTransportSecurity() bool { return true }
