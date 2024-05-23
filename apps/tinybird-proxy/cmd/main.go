package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/apps/tinybird-proxy/pkg/batch"
	"github.com/unkeyed/unkey/apps/tinybird-proxy/pkg/env"
	"github.com/unkeyed/unkey/apps/tinybird-proxy/pkg/id"
	"github.com/unkeyed/unkey/apps/tinybird-proxy/pkg/logging"
	"github.com/unkeyed/unkey/apps/tinybird-proxy/pkg/metrics"
	"github.com/unkeyed/unkey/apps/tinybird-proxy/pkg/tinybird"
)

var (
	port                      = env.String("PORT", "8080")
	tinybirdToken             = env.String("TINYBIRD_TOKEN")
	tinybirdBaseUrl           = env.String("TINYBIRD_BASE_URL", "https://api.tinybird.co")
	batchSize                 = env.Int64("BATCH_SIZE", 100000)
	bufferSize                = env.Int64("BUFFER_SIZE", 1000000)
	flushInterval             = env.Int64("FLUSH_INTERVAL", 1000)
	authorizationToken        = env.String("AUTHORIZATION_TOKEN", "")
	nodeId                    = env.String("NODE_ID", id.New(8))
	tinybirdMetricsDatasource = env.String("TINYBIRD_METRICS_DATASOURCE", "")
)

type event struct {
	datasource string
	row        any
}

func main() {

	logger := logging.New(nil)
	logger.Info().Str("nodeId", nodeId).Msg("Starting node")
	if authorizationToken == "" {

		fmt.Printf(`
=============================================================
No AUTHORIZATION_TOKEN provided, all requests will be allowed
=============================================================
		`)
	}

	tb := tinybird.New(tinybirdBaseUrl, tinybirdToken)

	m := metrics.New(nodeId)

	flush := func(ctx context.Context, events []event) {
		if len(events) == 0 {
			return
		}
		m.RecordFlush()
		logger.Info().Int("events", len(events)).Msg("Flushing")
		eventsByDatasource := map[string][]any{}
		for _, e := range events {
			if _, ok := eventsByDatasource[e.datasource]; !ok {
				eventsByDatasource[e.datasource] = []any{}
			}
			eventsByDatasource[e.datasource] = append(eventsByDatasource[e.datasource], e.row)

		}
		for datasource, rows := range eventsByDatasource {
			err := tb.Ingest(datasource, rows)
			if err != nil {
				logger.Err(err).Str("datasource", datasource).Int("rows", len(rows)).Msg("Error ingesting")
			}
		}
	}

	batcher := batch.New(batch.Config[event]{
		BatchSize:     batchSize,
		BufferSize:    bufferSize,
		FlushInterval: time.Duration(flushInterval) * time.Millisecond,
		Flush:         flush,
	})

	if tinybirdMetricsDatasource != "" {
		go m.PeriodicallyFlush(func(record metrics.Record) {
			logger.Info().Interface("record", record).Msg("Flushing metrics")
			batcher.Buffer(event{tinybirdMetricsDatasource, record})
		})
	}

	http.HandleFunc("/v0/events", func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		if (len(authorization) == 0 && len(authorizationToken) != 0) || subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(authorization, "Bearer ")), []byte(authorizationToken)) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		datasource := r.URL.Query().Get("name")
		if datasource == "" {
			http.Error(w, "missing ?name=", http.StatusBadRequest)
			return
		}

		dec := json.NewDecoder(r.Body)
		defer r.Body.Close()

		rows := []any{}

		for {
			var v any
			err := dec.Decode(&v)
			if err != nil {
				if err == io.EOF {
					break
				}
				logger.Err(err).Msg("Error decoding row")
				http.Error(w, err.Error(), http.StatusBadRequest)
			}
			rows = append(rows, v)
		}

		for _, row := range rows {
			batcher.Buffer(event{datasource, row})
		}
		m.RecordRequest()
		m.RecordRows(int64(len(rows)))

		response := tinybird.Response{
			SuccessfulRows:  len(rows),
			QuarantinedRows: 0,
		}
		responseBody, err := json.Marshal(response)
		if err != nil {
			logger.Err(err).Msg("Error marshalling response")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(responseBody)
		if err != nil {
			logger.Err(err).Msg("Error writing response")
		}

	})

	http.HandleFunc("/v1/liveness", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {

			logger.Err(err).Msg("Error writing response")
		}
	})

	server := &http.Server{
		Addr: fmt.Sprintf(":%s", port),
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := server.Shutdown(ctx)
		batcher.Close()

		if err != nil {
			logger.Err(err).Msg("Error shutting down")
			os.Exit(1)
		}
	}()

	logger.Info().Str("port", port).Msg("Listening")
	err := server.ListenAndServe()
	if err != nil {
		logger.Err(err).Msg("Error listening")
		os.Exit(1)
	}
}
