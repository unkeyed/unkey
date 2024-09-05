package eventrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/apps/agent/gen/openapi"
	"github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse"
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/prometheus"
	"github.com/unkeyed/unkey/apps/agent/pkg/tinybird"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type event struct {
	datasource string
	row        any
}

type Config struct {
	BatchSize     int
	BufferSize    int
	FlushInterval time.Duration

	Tinybird   *tinybird.Client
	Logger     logging.Logger
	Metrics    metrics.Metrics
	Clickhouse clickhouse.Bufferer
	AuthToken  string
}

type Service struct {
	logger     logging.Logger
	metrics    metrics.Metrics
	batcher    batch.BatchProcessor[event]
	tb         *tinybird.Client
	authToken  string
	clickhouse clickhouse.Bufferer
}

func New(config Config) (*Service, error) {

	flush := func(ctx context.Context, events []event) {
		if len(events) == 0 {
			return
		}
		// config.Metrics.RecordFlush()
		eventsByDatasource := map[string][]any{}
		for _, e := range events {
			if _, ok := eventsByDatasource[e.datasource]; !ok {
				eventsByDatasource[e.datasource] = []any{}
			}
			eventsByDatasource[e.datasource] = append(eventsByDatasource[e.datasource], e.row)
		}
		for datasource, rows := range eventsByDatasource {
			err := config.Tinybird.Ingest(datasource, rows)
			if err != nil {
				config.Logger.Err(err).Str("datasource", datasource).Int("rows", len(rows)).Msg("Error ingesting")
			}
			prometheus.EventRouterFlushedRows.With(map[string]string{
				"datasource": datasource,
			}).Add(float64(len(rows)))

			if datasource == "key_verifications__v2" {

				for _, row := range rows {
					e, ok := row.(tinybirdKeyVerification)
					if !ok {
						config.Logger.Error().Str("e", fmt.Sprintf("%T: %+v", row, row)).Msg("Error casting key verification")
						continue
					}
					config.Logger.Info().Interface("e", e).Msg("Key verification event")
					// dual write to clickhouse
					outcome := "VALID"
					if e.DeniedReason != "" {
						outcome = e.DeniedReason
					}
					config.Clickhouse.BufferKeyVerification(schema.KeyVerificationRequestV1{
						RequestID:   e.RequestID,
						Time:        e.Time,
						WorkspaceID: e.WorkspaceId,
						KeySpaceID:  e.KeySpaceId,
						KeyID:       e.KeyId,
						Region:      e.Region,
						Outcome:     outcome,
						IdentityID:  e.OwnerId,
					})
				}

			}

		}
	}

	batcher := batch.New(batch.Config[event]{
		BatchSize:     config.BatchSize,
		BufferSize:    config.BufferSize,
		FlushInterval: config.FlushInterval,
		Flush:         flush,
	})
	return &Service{
		logger:    config.Logger,
		metrics:   config.Metrics,
		batcher:   *batcher,
		tb:        config.Tinybird,
		authToken: config.AuthToken,
	}, nil
}

// this is what we currently send to tinybird
// we need to parse it and transform it into a clickhouse event, then dual write to both stores
type tinybirdKeyVerification struct {
	ApiId             string `json:"apiId"`
	EdgeRegion        string `json:"edgeRegion"`
	IpAddress         string `json:"ipAddress"`
	KeyId             string `json:"keyId"`
	Ratelimited       bool   `json:"ratelimited"`
	Region            string `json:"region"`
	RequestedResource string `json:"requestedResource"`
	Time              int64  `json:"time"`
	UsageExceeded     bool   `json:"usageExceeded"`
	UserAgent         string `json:"userAgent"`
	WorkspaceId       string `json:"workspaceId"`
	DeniedReason      string `json:"deniedReason,omitempty"`
	OwnerId           string `json:"ownerId,omitempty"`
	KeySpaceId        string `json:"keySpaceId,omitempty"`
	RequestID         string `json:"requestId,omitempty"`
	RequestBody       string `json:"requestBody,omitempty"`
	ResponeBody       string `json:"responseBody,omitempty"`
}

func (s *Service) CreateHandler() (string, http.HandlerFunc) {
	return "POST /v0/events", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		s.logger.Info().Msg("Received events")

		ctx, span := tracing.Start(r.Context(), tracing.NewSpanName("eventrouter", "v0/events"))
		defer span.End()

		err := auth.Authorize(ctx, s.authToken, r.Header.Get("Authorization"))
		if err != nil {
			s.logger.Warn().Err(err).Msg("failed to authorize request")
			w.WriteHeader(403)
			w.Write([]byte("Unauthorized"))
			return
		}

		datasource := r.URL.Query().Get("name")
		if datasource == "" {
			w.WriteHeader(400)
			w.Write([]byte("missing ?name= parameter"))
			return
		}

		successfulRows := 0
		switch datasource {
		case "key_verifications__v2":
			events, err := decode[tinybirdKeyVerification](r.Body)
			if err != nil {
				s.logger.Err(err).Msg("Error decoding request")
				w.WriteHeader(400)
				w.Write([]byte("Error decoding request"))
				return
			}
			for _, e := range events {
				s.batcher.Buffer(event{datasource, e})
			}
			successfulRows = len(events)
		default:
			events, err := decode[any](r.Body)
			if err != nil {
				s.logger.Err(err).Msg("Error decoding request")
				w.WriteHeader(400)
				w.Write([]byte("Error decoding request"))
				return
			}
			for _, e := range events {
				s.batcher.Buffer(event{datasource, e})
			}
			successfulRows = len(events)
		}

		response := openapi.V0EventsResponseBody{
			SuccessfulRows:  successfulRows,
			QuarantinedRows: 0,
		}

		b, err := json.Marshal(response)
		if err != nil {
			s.logger.Err(err).Msg("Error marshalling response")
			w.WriteHeader(500)
			w.Write([]byte("Error marshalling response"))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Write(b)

	}
}

// decode reads the body of the request and decodes it into a slice of T
// the reader will be closed automatically
func decode[T any](body io.ReadCloser) ([]T, error) {
	defer body.Close()

	dec := json.NewDecoder(body)

	rows := []T{}

	for {
		var v T
		err := dec.Decode(&v)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		rows = append(rows, v)
	}

	return rows, nil

}
