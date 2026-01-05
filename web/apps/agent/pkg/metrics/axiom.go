package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/svc/agent/pkg/batch"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/util"
)

type axiom struct {
	region  string
	nodeId  string
	batcher *batch.BatchProcessor[map[string]any]
}

type Config struct {
	Token   string
	NodeId  string
	Region  string
	Logger  logging.Logger
	Dataset string
}

func New(config Config) (*axiom, error) {

	client := http.DefaultClient

	batcher := batch.New(batch.Config[map[string]any]{
		BatchSize:     1000,
		FlushInterval: time.Second,
		BufferSize:    10000,
		Flush: func(ctx context.Context, batch []map[string]any) {
			buf, err := json.Marshal(batch)
			if err != nil {
				config.Logger.Err(err).Msg("failed to marshal events")
				return
			}

			req, err := http.NewRequest(
				http.MethodPost,
				fmt.Sprintf("https://api.axiom.co/v1/datasets/%s/ingest", config.Dataset),
				bytes.NewBuffer(buf),
			)
			if err != nil {
				config.Logger.Err(err).Msg("failed to create request")
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.Token))

			resp, err := client.Do(req)
			if err != nil {
				config.Logger.Err(err).Msg("failed to send request")
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					config.Logger.Err(err).Msg("failed to read response body")
					return
				}
				config.Logger.Error().Str("body", string(body)).Int("status", resp.StatusCode).Msg("failed to ingest events")
				return
			}

		},
	})
	a := &axiom{
		region:  config.Region,
		nodeId:  config.NodeId,
		batcher: batcher,
	}

	return a, nil
}

func (a *axiom) Close() {
	a.batcher.Close()
}

func (a *axiom) merge(m Metric, now time.Time) map[string]any {

	data := util.StructToMap(m)
	data["metric"] = m.Name()
	data["_time"] = now.UnixMilli()
	data["nodeId"] = a.nodeId
	data["region"] = a.region
	data["application"] = "agent"

	return data
}

func (a *axiom) Record(m Metric) {

	a.batcher.Buffer(a.merge(m, time.Now()))
}
