package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/unkeyed/sdks/api/go/v3/models/operations"
	"github.com/unkeyed/unkey/pkg/cli"
)

// SendBody decodes a JSON body and sends it using the generated SDK method.
func SendBody[Request, Response any](
	ctx context.Context,
	send func(context.Context, Request, ...operations.Option) (*Response, error),
	body string,
) (*Response, error) {
	request, err := decodeBody[Request](body)
	if err != nil {
		return nil, err
	}
	response, err := send(ctx, request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// RequestBody selects the typed --body request or the request built from flags.
func RequestBody[Request any](cmd *cli.Command, flags Request) (Request, error) {
	if cmd.FlagIsSet("body") {
		return decodeBody[Request](cmd.String("body"))
	}
	return flags, nil
}

func decodeBody[Request any](body string) (Request, error) {
	var request Request
	if strings.TrimSpace(body) == "null" {
		return request, fmt.Errorf("invalid JSON for --body: top-level null is not a request body")
	}

	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return request, fmt.Errorf("invalid JSON for --body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return request, fmt.Errorf("invalid JSON for --body: multiple JSON values")
		}
		return request, fmt.Errorf("invalid JSON for --body: %w", err)
	}
	return request, nil
}
