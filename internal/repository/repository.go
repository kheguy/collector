package repository

import (
	"encoding/json"
	"os"
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

func (s *MemStorage) Save(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	metrics := make([]models.Metrics, 0, len(s.data))
	for id, value := range s.data {
		metric := models.Metrics{ID: id}
		switch value := value.(type) {
		case int:
			delta := int64(value)
			metric.MType = models.Counter
			metric.Delta = &delta
		case float64:
			metric.MType = models.Gauge
			metric.Value = &value
		default:
			continue
		}
		metrics = append(metrics, metric)
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0666)
}

func (s *MemStorage) Restore(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, metric := range metrics {
		switch metric.MType {
		case models.Counter:
			if metric.Delta != nil {
				s.data[metric.ID] = int(*metric.Delta)
			}
		case models.Gauge:
			if metric.Value != nil {
				s.data[metric.ID] = *metric.Value
			}
		}
	}
	return nil
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
