package metrics

import (
	"context"
	"fmt"
	"time"

	ax "github.com/axiomhq/axiom-go/axiom"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type axiom struct {
	client  *ax.Client
	eventsC chan ax.Event
	region  string
}

type Config struct {
	Token   string
	Region  string
	Logger  logging.Logger
	Dataset string
}

func New(config Config) (Metrics, error) {

	client, err := ax.NewClient(
		ax.SetToken(config.Token),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create axiom client")
	}

	a := &axiom{
		client:  client,
		eventsC: make(chan ax.Event),
		region:  config.Region,
	}

	go func() {
		_, err := client.IngestChannel(context.Background(), config.Dataset, a.eventsC)
		if err != nil {
			config.Logger.Err(err).Msg("unable to ingest to axiom")
		}
	}()

	return a, nil
}

func (m *axiom) Close() {
	close(m.eventsC)
}

func (m *axiom) report(metric metricId, r any) {
	e := util.StructToMap(r)
	e["metric"] = metric
	e["_time"] = time.Now().UnixMilli()
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
