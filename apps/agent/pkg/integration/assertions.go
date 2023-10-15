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

type Asserter func(ctx context.Context, req AssertRequest) error

func AssertHeaderExists(key string) Asserter {
	return func(ctx context.Context, req AssertRequest) error {
		if req.Header.Get(key) == "" {
			return fmt.Errorf("header %s does not exist", key)
		}
		return nil
	}
}

func AssertHeaderEquals(key, value string) Asserter {
	return func(ctx context.Context, req AssertRequest) error {
		if req.Header.Get(key) != value {
			return fmt.Errorf("header %s does not equal %s", key, value)
		}
		return nil
	}
}

func AssertStatus(status int) Asserter {
	return func(ctx context.Context, req AssertRequest) error {
		if req.Status != status {
			return fmt.Errorf("status %d does not match %d", req.Status, status)
		}
		return nil
	}
}

func AssertBody[T comparable](expr string, value T) Asserter {
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

func AssertBodyExists(expr string) Asserter {
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
