package logging

import (
	"context"
	"encoding/json"

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

func NewAxiomWriter(client *ax.Client) *AxiomWriter {

	a := &AxiomWriter{
		eventsC: make(chan ax.Event),
	}

	go func() {
		_, err := client.IngestChannel(context.Background(), "events-logs", a.eventsC)
		if err != nil {
			log.Print("unable to ingest to axiom")
		}
	}()

	return a
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
