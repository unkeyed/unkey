package tinybird

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type Tinybird struct {
	token string

	client *http.Client

	keyVerificationsC chan analytics.KeyVerificationEvent
	closeC            chan struct{}

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
		token:             config.Token,
		client:            http.DefaultClient,
		keyVerificationsC: make(chan analytics.KeyVerificationEvent),
		closeC:            make(chan struct{}),
		logger:            config.Logger,
	}

	go t.consume()
	return t
}

func (t *Tinybird) consume() {

	for {
		select {
		case <-t.closeC:
			return
		case e := <-t.keyVerificationsC:

			err := t.publishEvent("key_verifications__v1", e)
			if err != nil {
				t.logger.Err(err).Msg("unable to publish v1 event to tinybird")
			}

			err = t.publishEvent("key_verifications__v2", e)
			if err != nil {
				t.logger.Err(err).Msg("unable to publish event to tinybird")
			}
		}
	}
}

func (t *Tinybird) Close() {
	t.closeC <- struct{}{}

}

func (t *Tinybird) PublishKeyVerificationEvent(ctx context.Context, event analytics.KeyVerificationEvent) {
	t.logger.Debug().Any("event", event).Msg("publishing event")
	t.keyVerificationsC <- event
}

func (t *Tinybird) publishEvent(datasource string, event interface{}) error {
	buf, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error marshalling event: %w", err)
	}

	body := bytes.NewBuffer(buf)
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

	return nil
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
