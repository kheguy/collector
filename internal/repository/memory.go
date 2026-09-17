package repository

import (
	"context"
	"sync"

	models "github.com/kheguy/collector/internal/model"
)

type MemoryStorage struct {
	data map[string]interface{}
	mu   sync.Mutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]interface{}),
	}
}

func (s *MemoryStorage) Get(name string, ctx context.Context) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.data[name], nil
}

func (s *MemoryStorage) Set(name string, mType string, value interface{}, ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch mType {
	case models.Gauge:
		s.data[name] = value.(float64)
	case models.Counter:
		if s.data[name] == nil {
			s.data[name] = 0
		}
		s.data[name] = s.data[name].(int) + value.(int)
	}
	return nil
}

func (s *MemoryStorage) GetAll(ctx context.Context) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]interface{})

	for key, value := range s.data {
		result[key] = value
	}

	return result, nil
}

func (s *MemoryStorage) SetAll(d map[string]interface{}, ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = d

	return nil
}

func (s *MemoryStorage) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]interface{})

	return nil
}
