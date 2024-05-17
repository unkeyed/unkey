package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"


	"github.com/unkeyed/unkey/apps/vault/pkg/logging"
)

// memory is an in-memory storage implementation for testing purposes.
type memory struct {
	config MemoryConfig
	sync.RWMutex
	data   map[string][]byte
	logger logging.Logger
}

type MemoryConfig struct {
	Logger logging.Logger
}

func NewMemory(config MemoryConfig) (Storage, error) {

	logger := config.Logger.With().Str("service", "storage").Logger()

	return &memory{config: config, logger: logger, data: make(map[string][]byte)}, nil
}

func (s *memory) Key(workspaceId string, dekID string) string {
	return fmt.Sprintf("%s/%s", workspaceId, dekID)
}

func (s *memory) Latest(workspaceId string) string {
	return s.Key(workspaceId, "LATEST")
}

func (s *memory) PutObject(ctx context.Context, key string, data []byte) error {

	s.Lock()
	defer s.Unlock()

	s.data[key] = data
	return nil
}

func (s *memory) GetObject(ctx context.Context, key string) ([]byte, error) {
	s.RLock()
	defer s.RUnlock()
	data := s.data[key]
	if data == nil {
		return nil, ErrObjectNotFound
	}
	return data, nil
}
func (s *memory) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	s.RLock()
	defer s.RUnlock()
	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		if prefix == "" || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil

}
