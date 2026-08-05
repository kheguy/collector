package repository

import (
	"sync"

	models "github.com/kheguy/collector/internal/model"
)

type MemStorage struct {
	/*
		[ISSUE] ПРИВАТНЫЕ ПОЛЯ ПАКЕТА С МАЛЕНЬКОЙ БУКФЫ!
	*/
	data map[string]interface{}
	mu   sync.Mutex
}

func MakeNewMemoryStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]interface{}),
	}
}

// Тут будем забирать метрику в будущем (logs?)
func (s *MemStorage) Get(name string) interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.data[name]
}

// Установка метрики
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

	// fmt.Printf("New value is set to %s for %s\n", value, name)
	// fmt.Printf("Store state is %v\n", s.Data)

	return s.data[name]
}

func (s *MemStorage) GetAll() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	/*
		[ISSUE] Вместо возврата ссылки копируем, так как может быть гонка из-аз изменений данных в другой горутине
		Возможно, стоит посмотреть в сторону RWMutex или снапшотов если данных много???
	*/
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
