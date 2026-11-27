package repository

import (
	"errors"

	models "github.com/kheguy/collector/internal/model"
)

// Вынес отдельно - не хотел повторять да и как будто тут еще что-то может быть
func batchMetricValue(metric models.Metrics) (interface{}, error) {
	if err := validateBatchMetric(metric); err != nil {
		return nil, err
	}
	if metric.MType == models.Gauge {
		return *metric.Value, nil
	}
	return int(*metric.Delta), nil
}

func validateBatchMetric(metric models.Metrics) error {
	if metric.ID == "" {
		return errors.New("metric name is empty")
	}

	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return errors.New("gauge has no value")
		}
	case models.Counter:
		if metric.Delta == nil {
			return errors.New("counter has no delta")
		}
		counter := int(*metric.Delta)
		if int64(counter) != *metric.Delta {
			return errors.New("counter overflows int")
		}
	default:
		return errors.New("unknown metric type")
	}
	return nil
}
