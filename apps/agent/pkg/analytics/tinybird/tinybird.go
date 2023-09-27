package tinybird

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type Tinybird struct {
	token string

	client *http.Client

	keyVerificationsC chan<- analytics.KeyVerificationEvent

	logger logging.Logger
}

var _ analytics.Analytics = &Tinybird{}

type Config struct {
	// Token is the Tinybird token
	Token string

	Logger logging.Logger
}

func New(config Config) *Tinybird {

	t := &Tinybird{
		token:  config.Token,
		client: http.DefaultClient,
		logger: config.Logger,
	}

	t.logger.Info().Msg("starting tinybird analytics")

	datasource := "key_verifications__v2"

	t.keyVerificationsC = batch.Process[analytics.KeyVerificationEvent](func(ctx context.Context, batch []analytics.KeyVerificationEvent) {
		items := make([]string, len(batch))
		for i, e := range batch {
			buf, err := json.Marshal(e)
			if err != nil {
				t.logger.Err(err).Msg("unable to marshal event")
				return
			}
			items[i] = string(buf)
		}

		body := bytes.NewBufferString(strings.Join(items, "\n"))
		req, err := http.NewRequest("POST", fmt.Sprintf("https://api.tinybird.co/v0/events?name=%s", datasource), body)
		if err != nil {
			t.logger.Err(err).Msg("error creating request")
			return
		}

		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.token))
		req.Header.Add("Content-Type", "application/json")

		resp, err := t.client.Do(req)
		if err != nil {
			t.logger.Err(err).Msg("error sending request")
			return
		}
		t.logger.Debug().Int("status", resp.StatusCode).Msg("sent request")
		defer resp.Body.Close()

		// LEGACY we're also sending the event to the old datasource

		req2, err := http.NewRequest("POST", fmt.Sprintf("https://api.tinybird.co/v0/events?name=%s", "key_verifications__v1"), body)
		if err != nil {
			t.logger.Err(err).Msg("error creating request")
			return
		}

		req2.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.token))
		req2.Header.Add("Content-Type", "application/json")

		resp2, err := t.client.Do(req2)
		if err != nil {
			t.logger.Err(err).Msg("error sending request")
			return
		}
		defer resp2.Body.Close()

	}, 1000, time.Second)

	return t
}

func (t *Tinybird) Close() {
	close(t.keyVerificationsC)
}

func (t *Tinybird) PublishKeyVerificationEvent(ctx context.Context, event analytics.KeyVerificationEvent) {
	t.logger.Debug().Any("event", event).Msg("publishing event")
	t.keyVerificationsC <- event
}

func (t *Tinybird) GetKeyStats(ctx context.Context, keyId string) (analytics.KeyStats, error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.tinybird.co/v0/pipes/x__endpoint_get_daily_key_stats.json?keyId=%s&token=%s", keyId, t.token), nil)
	if err != nil {
		return analytics.KeyStats{}, fmt.Errorf("unable to prepare request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return analytics.KeyStats{}, fmt.Errorf("unable to call tinybird: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return analytics.KeyStats{}, fmt.Errorf("unable to call tinybird: status=%d", res.StatusCode)
	}

	type tinybirdResponse struct {
		Data []struct {
			Time  string `json:"time"`
			Usage int64  `json:"usage"`
		} `json:"data"`
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return analytics.KeyStats{}, fmt.Errorf("unable to read body: %w", err)
	}
	tr := tinybirdResponse{}

	err = json.Unmarshal(body, &tr)
	if err != nil {
		return analytics.KeyStats{}, fmt.Errorf("unable to unmarshal body: %w", err)
	}

	stats := analytics.KeyStats{
		Usage: make([]analytics.ValueAtTime, len(tr.Data)),
	}
	for i, day := range tr.Data {
		t, err := time.Parse(time.DateTime, day.Time)
		if err != nil {
			return analytics.KeyStats{}, fmt.Errorf("unable to parse time %s: %w", day.Time, err)
		}
		stats.Usage[i] = analytics.ValueAtTime{
			Time:  t.UnixMilli(),
			Value: day.Usage,
		}
	}

	return stats, nil

}
