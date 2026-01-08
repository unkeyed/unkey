package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// memory is an in-memory storage implementation for testing purposes.
type memory struct {
	config MemoryConfig
	mu     sync.RWMutex
	data   map[string][]byte
	logger logging.Logger
}

type MemoryConfig struct {
	Logger logging.Logger
}

func NewMemory(config MemoryConfig) (Storage, error) {
	logger := config.Logger.With("service", "storage")

	return &memory{
		config: config,
		logger: logger,
		data:   make(map[string][]byte),
		mu:     sync.RWMutex{},
	}, nil
}

func (s *memory) Key(workspaceId string, dekID string) string {
	return fmt.Sprintf("%s/%s", workspaceId, dekID)
}

func (s *memory) Latest(workspaceId string) string {
	return s.Key(workspaceId, "LATEST")
}

func (s *memory) PutObject(ctx context.Context, key string, b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = b
	return nil
}

func (s *memory) GetObject(ctx context.Context, key string) ([]byte, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	b, ok := s.data[key]
	if !ok {
		return nil, false, nil
	}

	return b, true, nil
}

func (s *memory) ListObjectKeys(ctx context.Context, prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := []string{}
	for key := range s.data {
		if prefix == "" || !strings.HasPrefix(key, prefix) {
			continue
		}

		keys = append(keys, key)

	}
	return keys, nil
}
