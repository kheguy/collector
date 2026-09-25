package repository

import (
	"errors"

	models "github.com/kheguy/collector/internal/model"
)

// Вынес отдельно - не хотел повторять да и как будто тут еще что-то может быть
func batchMetricValue(metric models.Metrics) (interface{}, error) {
	if metric.ID == "" {
		return nil, errors.New("metric name is empty")
	}

	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return nil, errors.New("gauge has no value")
		}
		return *metric.Value, nil
	case models.Counter:
		if metric.Delta == nil {
			return nil, errors.New("counter has no delta")
		}
		counter := int(*metric.Delta)
		if int64(counter) != *metric.Delta {
			return nil, errors.New("counter overflows int")
		}
		return counter, nil
	default:
		return nil, errors.New("unknown metric type")
	}
}
