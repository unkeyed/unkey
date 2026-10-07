package httpdrain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unkeyed/unkey/svc/logdrain/sink"
)

func (a *Sink) deliverJSON(ctx context.Context, batch sink.Batch) (sink.Result, error) {
	records := make([]json.RawMessage, len(batch.Events))
	for i, event := range batch.Events {
		record, err := marshalRecord(event)
		if err != nil {
			return sink.Result{}, fmt.Errorf("marshal batch: %w", err)
		}
		records[i] = record
	}
	body, err := json.Marshal(records)
	if err != nil {
		return sink.Result{}, fmt.Errorf("marshal batch: %w", err)
	}
	result, _, err := a.post(ctx, batch, body, "application/json")
	if err != nil {
		return result, err
	}
	result.Acknowledged = result.HTTPStatus >= 200 && result.HTTPStatus < 300
	return result, nil
}
