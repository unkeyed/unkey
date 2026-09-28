package httpdrain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

func (a *Sink) deliverNDJSON(ctx context.Context, batch sink.Batch) (sink.Result, error) {
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	for _, event := range batch.Events {
		record, err := marshalRecord(event)
		if err != nil {
			return sink.Result{}, fmt.Errorf("marshal batch: %w", err)
		}
		if err := encoder.Encode(record); err != nil {
			return sink.Result{}, fmt.Errorf("marshal batch: %w", err)
		}
	}
	result, _, err := a.post(ctx, batch, body.Bytes(), "application/x-ndjson")
	if err != nil {
		return result, err
	}
	result.Acknowledged = result.HTTPStatus >= 200 && result.HTTPStatus < 300
	return result, nil
}
