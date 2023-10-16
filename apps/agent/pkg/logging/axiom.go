package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"log"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/batch"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type AxiomWriter struct {
	logsC chan<- map[string]any
}

type AxiomWriterConfig struct {
	AxiomToken string
}

func NewAxiomWriter(config AxiomWriterConfig) (*AxiomWriter, error) {

	dataset := "agent"
	client := http.DefaultClient

	logsC := batch.Process[map[string]any](func(ctx context.Context, b []map[string]any) {

		err := util.Retry(func() error {

			buf, err := json.Marshal(b)
			if err != nil {
				return fmt.Errorf("unable to marshal event: %s", err)
			}

			req, err := http.NewRequest("POST", fmt.Sprintf("https://api.axiom.co/v1/datasets/%s/ingest", dataset), bytes.NewBuffer(buf))
			if err != nil {
				return fmt.Errorf("unable to create request: %s", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AxiomToken))

			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("unable to create request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					return fmt.Errorf("unable to read response: %s", err)
				}
				return fmt.Errorf("axiom response status: %d: %s", resp.StatusCode, string(body))
			}
			return nil
		})
		if err != nil {
			log.Printf("unable to ingest to axiom: %s", err)
		}

	}, 1000, time.Second)

	a := &AxiomWriter{
		logsC: logsC,
	}
	return a, nil

}

func (aw *AxiomWriter) Close() {
	close(aw.logsC)
}

func (aw *AxiomWriter) Write(p []byte) (int, error) {
	e := make(map[string]any)

	err := json.Unmarshal(p, &e)
	if err != nil {
		return 0, err
	}
	e["_time"] = time.Now().UnixMilli()

	aw.logsC <- e
	return len(p), nil
}
