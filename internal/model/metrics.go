package models

const (
	// Counter обозначает целочисленный счётчик с накоплением значений
	Counter = "counter"
	// Gauge обозначает метрику с заменяемым значением float64
	Gauge = "gauge"
)

// MetricItem содержит имя и строковое значение метрики для HTML страницы
type MetricItem struct {
	Name  string
	Value string
}

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.

// Metrics описывает метрику в запросах JSON, ответах и файле хранилища
// Delta и Value указатели, чтобы отличать ноль от отсутствующего значения
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}
