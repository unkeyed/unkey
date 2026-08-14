package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/unkeyed/sdks/api/go/v2/models/operations"
)

// SendBody decodes a JSON body and sends it using the generated SDK method.
func SendBody[Request, Response any](
	ctx context.Context,
	send func(context.Context, Request, ...operations.Option) (*Response, error),
	body string,
) (*Response, error) {
	if strings.TrimSpace(body) == "null" {
		return nil, fmt.Errorf("invalid JSON for --body: top-level null is not a request body")
	}

	var request Request
	decoder := json.NewDecoder(strings.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return nil, fmt.Errorf("invalid JSON for --body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("invalid JSON for --body: multiple JSON values")
		}
		return nil, fmt.Errorf("invalid JSON for --body: %w", err)
	}

	response, err := send(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("%s", FormatError(err))
	}

	return response, nil
}
