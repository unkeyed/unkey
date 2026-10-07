package httpdrain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

type hecEvent struct {
	Time       float64         `json:"time"`
	Source     string          `json:"source"`
	Sourcetype string          `json:"sourcetype"`
	Event      json.RawMessage `json:"event"`
}

type hecResponse struct {
	Code *int `json:"code"`
}

func (a *Sink) deliverHEC(ctx context.Context, batch sink.Batch) (sink.Result, error) {
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	for _, event := range batch.Events {
		record, err := marshalRecord(event)
		if err != nil {
			return sink.Result{}, fmt.Errorf("marshal batch: %w", err)
		}
		if err := encoder.Encode(hecEvent{
			Time:       float64(event.Time) / 1000,
			Source:     "unkey",
			Sourcetype: event.Stream,
			Event:      record,
		}); err != nil {
			return sink.Result{}, fmt.Errorf("marshal batch: %w", err)
		}
	}
	result, truncated, err := a.post(ctx, batch, body.Bytes(), "application/json")
	if err != nil {
		return result, err
	}
	if truncated {
		return result, fmt.Errorf("HEC response exceeds diagnostic body limit")
	}
	result.Acknowledged = result.HTTPStatus >= 200 && result.HTTPStatus < 300
	if !result.Acknowledged || result.ResponseBody == "" {
		return result, nil
	}
	var response hecResponse
	if err := json.Unmarshal([]byte(result.ResponseBody), &response); err != nil {
		result.Acknowledged = false
		return result, fmt.Errorf("decode HEC response: %w", err)
	}
	result.Acknowledged = response.Code != nil && *response.Code == 0
	return result, nil
}
