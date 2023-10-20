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
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
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

	t.keyVerificationsC = batch.Process[analytics.KeyVerificationEvent](func(ctx context.Context, b []analytics.KeyVerificationEvent) {

		err := util.Retry(func() error {
			items := make([]string, len(b))
			for i, e := range b {
				buf, err := json.Marshal(e)
				if err != nil {
					return fmt.Errorf("unable to marshal event: %w", err)
				}
				items[i] = string(buf)
			}

			body := bytes.NewBufferString(strings.Join(items, "\n"))
			req, err := http.NewRequest("POST", fmt.Sprintf("https://api.tinybird.co/v0/events?name=%s", datasource), body)
			if err != nil {
				return fmt.Errorf("error creating request: %w", err)

			}

			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.token))
			req.Header.Add("Content-Type", "application/json")

			resp, err := t.client.Do(req)
			if err != nil {
				return fmt.Errorf("error sending request: %w", err)

			}
			defer resp.Body.Close()
			t.logger.Debug().Int("status", resp.StatusCode).Msg("sent request")
			return nil

		})
		if err != nil {
			t.logger.Error().Err(err).Msg("unable to send events to tinybird")
		}

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

func (t *Tinybird) GetKeyStats(ctx context.Context, workspaceId, apiId, keyId string) (analytics.KeyStats, error) {

	url := fmt.Sprintf("https://api.tinybird.co/v0/pipes/endpoint__get_daily_verifications__v1.json?workspaceId=%s&apiId=%s&keyId=%s&token=%s", workspaceId, apiId, keyId, t.token)
	req, err := http.NewRequest("GET", url, nil)
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
			Time          string `json:"time"`
			Success       int64  `json:"success"`
			RateLimited   int64  `json:"rateLimited"`
			UsageExceeded int64  `json:"usageExceeded"`
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
		Usage: make([]analytics.KeyUsage, len(tr.Data)),
	}
	for i, day := range tr.Data {
		t, err := time.Parse(time.DateTime, day.Time)
		if err != nil {
			return analytics.KeyStats{}, fmt.Errorf("unable to parse time %s: %w", day.Time, err)
		}
		stats.Usage[i] = analytics.KeyUsage{
			Time:          t.UnixMilli(),
			Success:       day.Success,
			RateLimited:   day.RateLimited,
			UsageExceeded: day.UsageExceeded,
		}
	}

	return stats, nil

}
