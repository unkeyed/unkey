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
	"go.uber.org/zap"
)

const topic = "key.changed"

type keyEventType string

var (
	KeyCreated keyEventType = "created"
	KeyUpdated keyEventType = "updated"
	KeyDeleted keyEventType = "deleted"
)

type KeyEvent struct {
	Type keyEventType `json:"type"`
	Key  struct {
		Id   string `json:"id"`
		Hash string `hson:"hash"`
	} `json:"key"`
}

type Kafka struct {
	sync.Mutex
	keyChangedReader *kafka.Reader
	keyChangedWriter *kafka.Writer

	callbackLock sync.RWMutex
	onKeyEvent   []func(ctx context.Context, e KeyEvent) error

	shutdownLock sync.RWMutex
	shutdown     bool
	logger       *zap.Logger
}

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
	return &Kafka{
		logger:       logger,
		callbackLock: sync.RWMutex{},
		keyChangedReader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{config.Broker},
			GroupID: config.GroupId,
			Topic:   topic,
			Dialer:  dialer,
		}),
		keyChangedWriter: kafka.NewWriter(kafka.WriterConfig{
			Brokers: []string{config.Broker},
			Topic:   topic,
			Dialer:  dialer,
		}),

		onKeyEvent:   make([]func(ctx context.Context, e KeyEvent) error, 0),
		shutdownLock: sync.RWMutex{},
		shutdown:     false,
	}, nil

}

func (k *Kafka) RegisterOnKeyEvent(handler func(ctx context.Context, e KeyEvent) error) {
	k.callbackLock.Lock()
	defer k.callbackLock.Unlock()
	k.onKeyEvent = append(k.onKeyEvent, handler)
}

func (k *Kafka) ProduceKeyEvent(ctx context.Context, eventType keyEventType, keyId, keyHash string) error {
	e := KeyEvent{
		Type: eventType,
	}
	e.Key.Id = keyId
	e.Key.Hash = keyHash
	value, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("unable to marshal KeyDeltedEvent: %w", err)
	}

	return k.keyChangedWriter.WriteMessages(ctx, kafka.Message{Value: value})
}

func (k *Kafka) Close() error {
	k.logger.Info("stopping..")
	defer k.logger.Info("stopped")
	k.Lock()
	defer k.Unlock()
	k.shutdownLock.Lock()
	k.shutdown = true
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

// Call Start in a goroutine
func (k *Kafka) Start() {
	for {
		k.shutdownLock.RLock()
		shutdown := k.shutdown
		k.shutdownLock.RUnlock()
		if shutdown {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
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
		e := KeyEvent{}
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
}
