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

	logsC := batch.Process[map[string]any](func(ctx context.Context, batch []map[string]any) {
		buf, err := json.Marshal(batch)
		if err != nil {
			log.Printf("unable to marshal event: %s", err)
			return
		}

		req, err := http.NewRequest("POST", fmt.Sprintf("https://api.axiom.co/v1/datasets/%s/ingest", dataset), bytes.NewBuffer(buf))
		if err != nil {
			log.Printf("unable to create request: %s", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.AxiomToken))

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("unable to create request: %s", err)
			return
		}
		if resp.StatusCode != 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("unable to read response: %s", err)
				return
			}
			log.Printf("unable to ingest to axiom: %s", string(body))
			return
		}
		resp.Body.Close()

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
