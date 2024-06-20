package tinybird

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func (c *Client) Ingest(datasource string, rows []string) error {

	body := strings.Join(rows, "\n")

	req, err := http.NewRequest("POST", c.baseUrl+"/v0/events?name="+datasource, strings.NewReader(body))
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

	res := Response{}
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

type Response struct {
	SuccessfulRows  int `json:"successful_rows"`
	QuarantinedRows int `json:"quarantined_rows"`
}
