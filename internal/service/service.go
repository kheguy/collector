package service

import (
	"context"
	"errors"
	"strconv"

	models "github.com/kheguy/collector/internal/model"
)

var (
	errParseFloat  = errors.New("can't parse gauge string to float64")
	errParseInt    = errors.New("can't parse counter string to int")
	errUnknownType = errors.New("unknown metric type")
)

type Storage interface {
	Get(name string, ctx context.Context) (interface{}, error)
	GetAll(ctx context.Context) (map[string]interface{}, error)
	Set(name string, mType string, value interface{}, ctx context.Context) error
	SetBatch(metrics []models.Metrics, ctx context.Context) error
	SetAll(data map[string]interface{}, ctx context.Context) error
}

type MetricsService struct {
	storage Storage
}

func MakeNewMetricsService(s Storage) *MetricsService {
	return &MetricsService{
		storage: s,
	}
}

func (s *MetricsService) GetMetric(name string, ctx context.Context) (string, error) {
	value, err := s.storage.Get(name, ctx)

	if err != nil {
		return "", err
	}

	switch v := value.(type) {
	case int:
		return strconv.Itoa(v), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	default:
		return "", nil
	}
}

func (s *MetricsService) GetRawMetric(name string, ctx context.Context) (interface{}, error) {
	return s.storage.Get(name, ctx)
}

func (s *MetricsService) GetAllMetrics(ctx context.Context) (map[string]interface{}, error) {
	return s.storage.GetAll(ctx)
}

func (s *MetricsService) UpdateMetrics(name string, typeOfValue string, value interface{}, ctx context.Context) error {

	var parsedValue interface{}

	switch typeOfValue {
	case models.Gauge:
		if str, ok := value.(string); ok {
			if val, err := strconv.ParseFloat(str, 64); err == nil {
				parsedValue = val
			} else {
				return errParseFloat
			}
		}

	case models.Counter:
		if str, ok := value.(string); ok {
			if val, err := strconv.Atoi(str); err == nil {
				parsedValue = val
			} else {
				return errParseInt
			}
		}
	default:
		return errUnknownType
	}

	err := s.storage.Set(name, typeOfValue, parsedValue, ctx)

	return err
}

func (s *MetricsService) UpdateMetricsBatch(metrics []models.Metrics, ctx context.Context) error {
	return s.storage.SetBatch(metrics, ctx)
}
