package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
	"go.uber.org/zap"
)

const topic = "key.changed"

type Kafka struct {
	sync.Mutex
	keyChangedReader *kafka.Reader
	keyChangedWriter *kafka.Writer

	callbackLock sync.RWMutex
	onKeyEvent   []func(ctx context.Context, e events.KeyEvent) error

	shutdownLock sync.RWMutex
	shutdown     bool
	shutdownC    chan struct{}
	logger       *zap.Logger

	// Events are first written to this channel and then flushed to kafka
	// This allows much cleaner code for users of this package
	keyEventBuffer chan events.KeyEvent
}

// static checking
var _ events.EventBus = &Kafka{}

type Config struct {
	GroupId  string
	Broker   string
	Username string
	Password string
	Logger   *zap.Logger
}

func New(config Config) (*Kafka, error) {
	logger := config.Logger.With(zap.String("pkg", "kafka"))
	logger.Info("starting kafka")
	mechanism, err := scram.Mechanism(scram.SHA256, config.Username, config.Password)
	if err != nil {
		return nil, fmt.Errorf("unable to create scram mechanism: %w", err)
	}

	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		TLS:           &tls.Config{},
		SASLMechanism: mechanism,
	}
	k := &Kafka{
		logger:       logger,
		callbackLock: sync.RWMutex{},
		keyChangedReader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{config.Broker},
			GroupID:     config.GroupId,
			Topic:       topic,
			Dialer:      dialer,
			StartOffset: kafka.LastOffset,
		}),
		keyChangedWriter: kafka.NewWriter(kafka.WriterConfig{
			Brokers: []string{config.Broker},
			Topic:   topic,
			Dialer:  dialer,
		}),

		onKeyEvent:     make([]func(ctx context.Context, e events.KeyEvent) error, 0),
		shutdownLock:   sync.RWMutex{},
		shutdown:       false,
		shutdownC:      make(chan struct{}),
		keyEventBuffer: make(chan events.KeyEvent, 1024),
	}

	return k, nil

}

func (k *Kafka) OnKeyEvent(handler func(ctx context.Context, e events.KeyEvent) error) {
	k.callbackLock.Lock()
	defer k.callbackLock.Unlock()
	k.onKeyEvent = append(k.onKeyEvent, handler)

}

// EmitKeyEvent writes to a channel and returns immediately, it does not wait for acks from kafka
func (k *Kafka) EmitKeyEvent(ctx context.Context, e events.KeyEvent) {
	k.keyEventBuffer <- e
}

func (k *Kafka) Close() error {
	k.logger.Info("stopping..")
	defer k.logger.Info("stopped")
	k.Lock()
	defer k.Unlock()
	k.shutdownLock.Lock()
	k.shutdown = true
	k.shutdownC <- struct{}{}
	k.shutdownLock.Unlock()

	k.logger.Info("stopping reader")
	err := k.keyChangedReader.Close()
	if err != nil {
		return err
	}

	k.logger.Info("stopping writer")
	err = k.keyChangedWriter.Close()
	if err != nil {
		return err
	}
	return nil
}

// Starts all goroutines
func (k *Kafka) Start() {

	go func() {
		for {
			select {
			case <-k.shutdownC:
				return
			case e := <-k.keyEventBuffer:
				value, err := json.Marshal(e)
				if err != nil {
					k.logger.Error("unable to marshal KeyEvent", zap.Error(err), zap.String("type", string(e.Type)), zap.String("keyId", e.Key.Id))
					continue
				}

				err = k.keyChangedWriter.WriteMessages(context.Background(), kafka.Message{Value: value})
				if err != nil {
					k.logger.Error("unable write messages to kafka", zap.Error(err), zap.String("type", string(e.Type)), zap.String("keyId", e.Key.Id))
					continue
				}
			}
		}

	}()
	go func() {
		for {
			k.shutdownLock.RLock()
			shutdown := k.shutdown
			k.shutdownLock.RUnlock()
			if shutdown {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
			defer cancel()
			m, err := k.keyChangedReader.FetchMessage(ctx)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				k.logger.Error("unable to fetch message", zap.Error(err))
				continue
			}

			if len(m.Value) == 0 {
				k.logger.Warn("message is empty", zap.String("topic", m.Topic))
				continue
			}
			e := events.KeyEvent{}
			err = json.Unmarshal(m.Value, &e)
			if err != nil {
				k.logger.Error("unable to unmarshal message", zap.Error(err), zap.String("value", string(m.Value)))
				continue
			}
			k.callbackLock.RLock()
			for _, handler := range k.onKeyEvent {
				err := handler(ctx, e)
				if err != nil {
					k.logger.Error("unable to handle message", zap.Error(err))
					k.callbackLock.RUnlock()
					continue
				}
			}
			k.callbackLock.RUnlock()

			err = k.keyChangedReader.CommitMessages(ctx, m)
			if err != nil {
				k.logger.Error("unable to commit message", zap.Error(err))
				continue
			}

		}
	}()
}
