package repository

import (
	"encoding/json"
	"os"

	models "github.com/kheguy/collector/internal/model"
)

type FileProcessor struct {
	path string
}

func NewFileProcessor(p string) *FileProcessor {
	return &FileProcessor{
		path: p,
	}
}

func (f *FileProcessor) Save(d map[string]interface{}) error {
	metrics := make([]models.Metrics, 0, len(d))

	for id, value := range d {
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
	return os.WriteFile(f.path, data, 0666)
}

func (f *FileProcessor) Restore() (map[string]interface{}, error) {
	d := make(map[string]interface{})
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	if len(data) == 0 {
		return d, nil
	}

	var metrics []models.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return d, err
	}

	for _, metric := range metrics {
		switch metric.MType {
		case models.Counter:
			if metric.Delta != nil {
				d[metric.ID] = int(*metric.Delta)
			}
		case models.Gauge:
			if metric.Value != nil {
				d[metric.ID] = *metric.Value
			}
		}
	}
	return d, nil
}
