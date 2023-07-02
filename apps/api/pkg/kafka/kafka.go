package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/scram"
	"go.uber.org/zap"
	"sync"
	"time"
)

const keyDeletedTopic = "key.deleted"

type KeyDeletedEvent struct {
	Key struct {
		Id   string `json:"id"`
		Hash string `hson:"hash"`
	} `json:"key"`
}

type Kafka struct {
	keyDeletedReader *kafka.Reader
	keyDeletedWriter *kafka.Writer

	callbackLock sync.RWMutex
	onKeyDeleted []func(e KeyDeletedEvent) error

	logger *zap.Logger
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
	logger.Info("starting kafka", zap.String("username", config.Username), zap.String("password", config.Password))
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
		keyDeletedReader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{config.Broker},
			GroupID: config.GroupId,
			Topic:   keyDeletedTopic,
			Dialer:  dialer,
		}),
		keyDeletedWriter: kafka.NewWriter(kafka.WriterConfig{
			Brokers: []string{config.Broker},
			Topic:   keyDeletedTopic,
			Dialer:  dialer,
		}),

		onKeyDeleted: make([]func(e KeyDeletedEvent) error, 0),
	}, nil

}

func (k *Kafka) RegisterOnKeyDeleted(handler func(e KeyDeletedEvent) error) {
	k.callbackLock.Lock()
	defer k.callbackLock.Unlock()
	k.onKeyDeleted = append(k.onKeyDeleted, handler)
}

func (k *Kafka) ProduceKeyDeletedEvent(ctx context.Context, keyId, keyHash string) error {
	e := KeyDeletedEvent{}
	e.Key.Id = keyId
	e.Key.Hash = keyHash
	value, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("unable to marshal KeyDeltedEvent: %w", err)
	}
	return k.keyDeletedWriter.WriteMessages(ctx, kafka.Message{Value: value})
}

// Call Start in a goroutine
func (k *Kafka) Start() {
	for {
		ctx := context.Background()
		m, err := k.keyDeletedReader.FetchMessage(ctx)
		if err != nil {
			k.logger.Error("unable to fetch message", zap.Error(err))
			continue
		}
		k.logger.Debug("consuming message", zap.String("topic", m.Topic), zap.Int64("offset", m.Offset))

		if len(m.Value) == 0 {
			k.logger.Warn("message is empty", zap.String("topic", m.Topic))
			continue
		}
		e := KeyDeletedEvent{}
		err = json.Unmarshal(m.Value, &e)
		if err != nil {
			k.logger.Error("unable to unmarshal message", zap.Error(err), zap.String("value", string(m.Value)))
			continue
		}
		k.callbackLock.RLock()
		for _, handler := range k.onKeyDeleted {
			err := handler(e)
			if err != nil {
				k.logger.Error("unable to handle message", zap.Error(err))
				continue
			}
		}
		k.callbackLock.RUnlock()

		err = k.keyDeletedReader.CommitMessages(ctx, m)
		if err != nil {
			k.logger.Error("unable to commit message", zap.Error(err))
			continue
		}

	}
}
