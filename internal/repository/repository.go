package repository

import (
	models "github.com/kheguy/collector/internal/model"
)

type MemStorage struct {
	Data map[string]interface{}
}

func MakeNewMemoryStorage() *MemStorage {
	return &MemStorage{
		Data: make(map[string]interface{}),
	}
}

// Тут будем забирать метрику в будущем (logs?)
func (s *MemStorage) Get(name string) interface{} {
	return s.Data[name]
}

// Установка метрики
func (s *MemStorage) Set(name string, mType string, value interface{}) interface{} {
	switch mType {
	case models.Gauge:
		s.Data[name] = value.(float64)
	case models.Counter:
		if s.Data[name] == nil {
			s.Data[name] = 0
		}
		s.Data[name] = value.(int)
	}

	// fmt.Printf("New value is set to %s for %s\n", value, name)
	// fmt.Printf("Store state is %v\n", s.Data)

	return s.Data[name]
}

func (s *MemStorage) GetAll() map[string]interface{} {
	return s.Data
}
