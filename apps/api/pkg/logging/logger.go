package logging

import (
	"log"

	adapter "github.com/axiomhq/axiom-go/adapters/zap"
	"go.uber.org/zap"
)

type Logger = *zap.Logger

func New(pretty ...bool) Logger {

	core, err := adapter.New(adapter.SetDataset("api-go"))
	if err != nil {

		logger, zapErr := zap.NewDevelopment()
		if zapErr != nil {
			log.Fatalf("can't initialize logger: %s", err.Error())
		}
		return logger
	}
	logger := zap.New(core)
	logger.Info("logging set up with axiom")
	return logger

}

func NewNoopLogger() Logger {
	return zap.NewNop()
}
