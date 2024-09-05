package tinybird

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/apps/agent/pkg/openapi"
)

type Client struct {
	baseUrl    string
	token      string
	httpClient *http.Client
}

func New(baseUrl string, token string) *Client {
	return &Client{
		baseUrl:    baseUrl,
		token:      token,
		httpClient: http.DefaultClient,
	}
}

func (c *Client) Ingest(datasource string, rows []any) error {

	body := ""
	for _, row := range rows {
		str, err := json.Marshal(row)
		if err != nil {
			return err
		}
		body += string(str) + "\n"
	}

	req, err := http.NewRequest(http.MethodPost, c.baseUrl+"/v0/events?name="+datasource, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error performing POST request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("error ingesting rows, status code: " + resp.Status)

	}

	res := openapi.V0EventsResponseBody{}
	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)

	}
	err = json.Unmarshal(resBody, &res)
	if err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}
	if res.SuccessfulRows != len(rows) {
		return errors.New("error ingesting all rows")
	}

	return nil
}
