package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Step[T any] struct {
	Name   string
	Body   map[string]any
	Url    string
	Method string
	Header map[string]string

	Assertions []assertion
}

func (s Step[T]) fail(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	log.Println()
	log.Println()
	log.Printf("Step: %s failed: %s\n", s.Name, msg)
	os.Exit(1)
}

func (s Step[T]) Run(ctx context.Context, res T) T {
	id := getStepId()

	start := time.Now()
	log.Printf("[%03d] - Step: %s ...", id, s.Name)
	if len(s.Assertions) == 0 {
		log.Printf(" no assertions ...")
	}
	defer func() {
		log.Printf("        done (%s)\n", time.Since(start).Round(time.Millisecond))
	}()
	requestBody, err := json.Marshal(s.Body)
	if err != nil {
		s.fail("unable to marshal body", err)
	}
	req, err := http.NewRequest(s.Method, s.Url, bytes.NewBuffer(requestBody))
	if err != nil {
		s.fail("unable to create request", err)
	}
	for k, v := range s.Header {
		req.Header.Set(k, v)
	}

	httpResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		s.fail("unable to make request", err)
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		s.fail("unable to read response body", err)
	}

	for _, assertion := range s.Assertions {
		err := assertion(ctx, AssertRequest{
			Status: httpResponse.StatusCode,
			Header: httpResponse.Header,
			Body:   string(body),
		})
		if err != nil {
			s.fail("An assertion failed for request %s %s: %w. Got status: %d - %s", s.Method, s.Url, err, httpResponse.StatusCode, string(body))
		}
	}

	err = json.Unmarshal(body, &res)
	if err != nil {
		s.fail("unable to unmarshal response: %w", err)
	}

	return res

}
