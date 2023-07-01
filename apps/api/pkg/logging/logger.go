package logging

import (
	adapter "github.com/axiomhq/axiom-go/adapters/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"time"
)

type Logger = *zap.Logger

func New() Logger {

	axiomCore, err := adapter.New(adapter.SetDataset("api"))
	if err != nil {

		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, zapErr := config.Build()
		if zapErr != nil {
			log.Fatalf("can't initialize logger: %s", err.Error())
		}
		logger.Info("new development logger created")
		return logger
	}
	go func() {
		for range time.NewTicker(time.Second * 5).C {

			err = axiomCore.Sync()
			if err != nil {
				log.Printf("can't sync to axiom: %s\n", err.Error())
			}
		}

	}()

	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		os.Stdout,
		zapcore.DebugLevel,
	)

	logger := zap.New(zapcore.NewTee(consoleCore, axiomCore))

	logger.Info("logging set up with axiom")

	return logger

}

func NewNoopLogger() Logger {
	return zap.NewNop()
}
