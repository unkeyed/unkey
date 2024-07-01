package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type axiom struct {
	eventsC chan<- map[string]any
	region  string
	nodeId  string
}

type Config struct {
	Token   string
	NodeId  string
	Region  string
	Logger  logging.Logger
	Dataset string
}

func New(config Config) (Metrics, error) {

	client := http.DefaultClient

	eventsC := batch.Process[map[string]any](func(ctx context.Context, events []map[string]any) {

		buf, err := json.Marshal(events)
		if err != nil {
			config.Logger.Err(err).Msg("failed to marshal events")
			return
		}

		req, err := http.NewRequest("POST", fmt.Sprintf("https://api.axiom.co/v1/datasets/%s/ingest", config.Dataset), bytes.NewBuffer(buf))
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

	}, 1000, time.Second)

	a := &axiom{
		eventsC: eventsC,
		region:  config.Region,
		nodeId:  config.NodeId,
	}

	return a, nil
}

func (m *axiom) Close() {
	close(m.eventsC)
}

func (m *axiom) report(metric metricId, r any) {
	e := util.StructToMap(r)
	e["metric"] = metric
	e["_time"] = time.Now().UnixMilli()
	e["nodeId"] = m.nodeId
	e["region"] = m.region
	e["application"] = "agent"
	m.eventsC <- e
}

func (m *axiom) ReportCacheHealth(r CacheHealthReport) {
	m.report(cacheHealth, r)
}

func (m *axiom) ReportDatabaseLatency(r DatabaseLatencyReport) {
	m.report(databaseLatency, r)
}

func (m *axiom) ReportCacheHit(r CacheHitReport) {
	m.report(cacheHit, r)
}

func (m *axiom) ReportSystemLoad(r SystemLoadReport) {
	m.report(systemLoad, r)
}
