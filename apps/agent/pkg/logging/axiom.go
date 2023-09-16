package logging

import (
	"context"
	"encoding/json"
	"fmt"

	"log"
	"time"

	ax "github.com/axiomhq/axiom-go/axiom"
)

type AxiomWriter struct {
	eventsC chan ax.Event
}

type AxiomWriterConfig struct {
	AxiomToken string
	AxiomOrgId string
}

func NewAxiomWriter(config AxiomWriterConfig) (*AxiomWriter, error) {

	client, err := ax.NewClient(
		ax.SetPersonalTokenConfig(config.AxiomToken, config.AxiomOrgId),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create axiom client")
	}
	a := &AxiomWriter{
		eventsC: make(chan ax.Event),
	}

	go func() {
		_, err := client.IngestChannel(context.Background(), "agent", a.eventsC)
		if err != nil {
			log.Print("unable to ingest to axiom")
		}
	}()

	return a, nil
}

func (aw *AxiomWriter) Close() {
	close(aw.eventsC)
}

func (aw *AxiomWriter) Write(p []byte) (int, error) {
	e := make(map[string]any)

	err := json.Unmarshal(p, &e)
	if err != nil {
		return 0, err
	}
	e["_time"] = time.Now().UnixMilli()

	aw.eventsC <- e
	return len(p), nil
}
