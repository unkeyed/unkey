package tinybird

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"net/http"
)

type Tinybird struct {
	token string

	client *http.Client

	keyVerificationsC chan KeyVerificationEvent
	closeC            chan struct{}

	logger *zap.Logger
}

type Config struct {
	// Token is the Tinybird token
	Token string

	Logger *zap.Logger
}

func New(config Config) *Tinybird {

	t := &Tinybird{
		token:             config.Token,
		client:            http.DefaultClient,
		keyVerificationsC: make(chan KeyVerificationEvent),
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
				t.logger.Error("unable to publish event to tinybird", zap.Error(err))
			}
		}
	}
}

func (t *Tinybird) Close() {
	t.closeC <- struct{}{}

}

type KeyVerificationEvent struct {
	ApiId       string `json:"apiId"`
	WorkspaceId string `json:"workspaceId"`
	KeyId       string `json:"keyId"`
	Ratelimited bool   `json:"ratelimited"`
	Time        int64  `json:"time"`
}

func (t *Tinybird) PublishKeyVerificationEventChannel() chan<- KeyVerificationEvent {
	return t.keyVerificationsC
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
