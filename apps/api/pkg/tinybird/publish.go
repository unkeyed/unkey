package tinybird

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Tinybird struct {
	token string

	client *http.Client
}

type Config struct {
	// Token is the Tinybird token
	Token string
}

func New(config Config) *Tinybird {

	return &Tinybird{
		token:  config.Token,
		client: http.DefaultClient,
	}
}

type KeyVerificationEvent struct {
	ApiId       string `json:"apiId"`
	WorkspaceId string `json:"workspaceId"`
	KeyId       string `json:"keyId"`
	Ratelimited bool   `json:"ratelimited"`
	Time        int64  `json:"time"`
}

func (t *Tinybird) PublishKeyVerificationEvent(datasource string, event KeyVerificationEvent) error {
	return t.publishEvent(datasource, event)
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
