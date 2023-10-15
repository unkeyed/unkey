package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PaesslerAG/gval"
)

type AssertRequest struct {
	Status int
	Header http.Header
	Body   string
}

type assertion func(ctx context.Context, req AssertRequest) error

func assertHeaderExists(key string) assertion {
	return func(ctx context.Context, req AssertRequest) error {
		if req.Header.Get(key) == "" {
			return fmt.Errorf("header %s does not exist", key)
		}
		return nil
	}
}

func assertStatus(status int) assertion {
	return func(ctx context.Context, req AssertRequest) error {
		if req.Status != status {
			return fmt.Errorf("status %d does not match %d", req.Status, status)
		}
		return nil
	}
}

func assertBody[T comparable](expr string, value T) assertion {
	return func(ctx context.Context, req AssertRequest) error {
		var data interface{}
		err := json.Unmarshal([]byte(req.Body), &data)
		if err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}

		actual, err := gval.Evaluate(expr, data)
		if err != nil {
			return fmt.Errorf("unable to evaluate expression %s: %w", expr, err)
		}

		if actual != value {
			return fmt.Errorf("expr %s does not equal (%T:%#v), got: (%T:%#v)", expr, value, value, actual, actual)
		}
		return nil
	}
}

func assertBodyExists(expr string) assertion {
	return func(ctx context.Context, req AssertRequest) error {
		var data interface{}
		err := json.Unmarshal([]byte(req.Body), &data)
		if err != nil {
			return fmt.Errorf("unable to unmarshal json: %w", err)
		}
		actual, err := gval.Evaluate(expr, data)
		if err != nil {
			return fmt.Errorf("unable to evaluate expression %s: %w", expr, err)
		}

		if actual == nil {
			return fmt.Errorf("expr %s does not exist", expr)
		}

		return nil
	}
}
