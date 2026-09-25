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

func (s *MemoryStorage) Ping(context.Context) error {
	return nil
}

func (s *MemoryStorage) Get(ctx context.Context, name string) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.data[name], nil
}

func (s *MemoryStorage) Set(ctx context.Context, name string, mType string, value interface{}) error {
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

func (s *MemoryStorage) SetBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	values := make([]interface{}, len(metrics))
	for i, metric := range metrics {
		value, err := batchMetricValue(metric)
		if err != nil {
			return err
		}
		values[i] = value
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			s.data[metric.ID] = values[i].(float64)
		case models.Counter:
			current, _ := s.data[metric.ID].(int)
			s.data[metric.ID] = current + values[i].(int)
		}
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

func (s *MemoryStorage) SetAll(ctx context.Context, d map[string]interface{}) error {
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
