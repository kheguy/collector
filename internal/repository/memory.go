package repository

import (
	"context"
	"sync"

	models "github.com/kheguy/collector/internal/model"
)

// MemoryStorage хранит метрики в памяти
type MemoryStorage struct {
	data map[string]interface{}
	mu   sync.Mutex
}

// NewMemoryStorage создаёт пустое хранилище метрик
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		data: make(map[string]interface{}),
	}
}

// Ping всегда возвращает nil
func (s *MemoryStorage) Ping(context.Context) error {
	return nil
}

// Get возвращает значение метрики по имени, если метрики нет возвращает nil
func (s *MemoryStorage) Get(ctx context.Context, name string) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.data[name], nil
}

// Set заменяет значение gauge или прибавляет значение к counter
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

// SetBatch проверяет весь пакет и применяет его под одной блокировкой
// При ошибке проверки метрики не меняются
func (s *MemoryStorage) SetBatch(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	for _, metric := range metrics {
		if err := validateBatchMetric(metric); err != nil {
			return err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			s.data[metric.ID] = *metric.Value
		case models.Counter:
			current, _ := s.data[metric.ID].(int)
			s.data[metric.ID] = current + int(*metric.Delta)
		}
	}

	return nil
}

// GetAll возвращает копию карты метрик
func (s *MemoryStorage) GetAll(ctx context.Context) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]interface{}, len(s.data))

	for key, value := range s.data {
		result[key] = value
	}

	return result, nil
}

// TakeAll возвращает текущую карту метрик, заменяя её в хранилище пустой
func (s *MemoryStorage) TakeAll(ctx context.Context) (map[string]interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := s.data
	s.data = make(map[string]interface{})

	return result, nil
}

// Restore возвращает неотправленные метрики, счётчики суммируются, текущие значения gauge сохраняются
func (s *MemoryStorage) Restore(ctx context.Context, data map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, value := range data {
		switch value := value.(type) {
		case int:
			current, _ := s.data[name].(int)
			s.data[name] = current + value
		case float64:
			if _, exists := s.data[name]; !exists {
				s.data[name] = value
			}
		}
	}

	return nil
}

// SetAll заменяет все метрики переданной картой без копирования
func (s *MemoryStorage) SetAll(ctx context.Context, d map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = d

	return nil
}

// Clear удаляет все метрики
func (s *MemoryStorage) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data = make(map[string]interface{})

	return nil
}
