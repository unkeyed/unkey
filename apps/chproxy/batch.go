package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type Batch struct {
	Rows   []string
	Params url.Values
	Table  string
}

func persist(ctx context.Context, batch *Batch, config *Config) error {
	ctx, span := telemetry.Tracer.Start(ctx, batch.Table)
	defer span.End()

	if len(batch.Rows) == 0 {
		return nil
	}

	telemetry.Metrics.BatchCounter.Add(ctx, 1)
	telemetry.Metrics.RowCounter.Add(ctx, int64(len(batch.Rows)))

	span.SetAttributes(
		attribute.Int("rows", len(batch.Rows)),
	)

	u, err := url.Parse(config.ClickhouseURL)
	if err != nil {
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	u.RawQuery = batch.Params.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), strings.NewReader(strings.Join(batch.Rows, "\n")))
	if err != nil {
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	req.Header.Add("Content-Type", "text/plain")

	username := u.User.Username()

	password, ok := u.User.Password()
	if !ok {
		err := fmt.Errorf("password not set")
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	req.SetBasicAuth(username, password)

	res, err := httpClient.Do(req)
	if err != nil {
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		telemetry.Metrics.ErrorCounter.Add(ctx, 1)
		body, err := io.ReadAll(res.Body)
		if err != nil {
			config.Logger.Error("error reading body",
				"error", err)

			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}

		errorMsg := string(body)

		config.Logger.Error("unable to persist batch",
			"response", errorMsg,
			"status_code", res.StatusCode,
			"query", batch.Params.Get("query"))

		span.SetStatus(codes.Error, errorMsg)
		span.RecordError(fmt.Errorf("HTTP %d: %s", res.StatusCode, errorMsg))

		return fmt.Errorf("http error: %v", errorMsg)
	}

	config.Logger.Info("rows persisted",
		"count", len(batch.Rows),
		"table", batch.Table)

	span.SetStatus(codes.Ok, "")

	return nil
}
