package repository

import (
	"sync"

	models "github.com/kheguy/collector/internal/model"
)

type MemStorage struct {
	data map[string]interface{}
	mu   sync.Mutex
}

func MakeNewMemoryStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]interface{}),
	}
}

func (s *MemStorage) Get(name string) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.data[name]
}

func (s *MemStorage) Set(name string, mType string, value interface{}) interface{} {
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

	return s.data[name]
}

func (s *MemStorage) GetAll() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]interface{})

	for key, value := range s.data {
		result[key] = value
	}

	return result
}

func (s *MemStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]interface{})
}
