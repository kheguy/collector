package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	models "github.com/kheguy/collector/internal/model"
)

var (
	ErrInvalidMetric = errors.New("invalid metric")
	errParseFloat    = errors.New("can't parse gauge string to float64")
	errParseInt      = errors.New("can't parse counter string to int")
	errUnknownType   = errors.New("unknown metric type")
)

type Storage interface {
	Get(ctx context.Context, name string) (interface{}, error)
	GetAll(ctx context.Context) (map[string]interface{}, error)
	Set(ctx context.Context, name string, mType string, value interface{}) error
	SetBatch(ctx context.Context, metrics []models.Metrics) error
	SetAll(ctx context.Context, data map[string]interface{}) error
}

type MetricsService struct {
	storage Storage
}

func MakeNewMetricsService(s Storage) *MetricsService {
	return &MetricsService{
		storage: s,
	}
}

func (s *MetricsService) GetMetric(ctx context.Context, name string) (string, error) {
	value, err := s.storage.Get(ctx, name)

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

func (s *MetricsService) GetRawMetric(ctx context.Context, name string) (interface{}, error) {
	return s.storage.Get(ctx, name)
}

func (s *MetricsService) GetAllMetrics(ctx context.Context) (map[string]interface{}, error) {
	return s.storage.GetAll(ctx)
}

func (s *MetricsService) UpdateMetrics(ctx context.Context, name string, typeOfValue string, value interface{}) error {

	var parsedValue interface{}

	switch typeOfValue {
	case models.Gauge:
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("%w: %w", ErrInvalidMetric, errParseFloat)
		}
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidMetric, errParseFloat)
		}
		parsedValue = val

	case models.Counter:
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("%w: %w", ErrInvalidMetric, errParseInt)
		}
		val, err := strconv.Atoi(str)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidMetric, errParseInt)
		}
		parsedValue = val
	default:
		return fmt.Errorf("%w: %w", ErrInvalidMetric, errUnknownType)
	}

	err := s.storage.Set(ctx, name, typeOfValue, parsedValue)

	return err
}

func (s *MetricsService) UpdateMetricsBatch(ctx context.Context, metrics []models.Metrics) error {
	for _, metric := range metrics {
		if metric.ID == "" {
			return fmt.Errorf("%w: metric name is empty", ErrInvalidMetric)
		}
		switch metric.MType {
		case models.Gauge:
			if metric.Value == nil {
				return fmt.Errorf("%w: gauge has no value", ErrInvalidMetric)
			}
		case models.Counter:
			if metric.Delta == nil {
				return fmt.Errorf("%w: counter has no delta", ErrInvalidMetric)
			}
		default:
			return fmt.Errorf("%w: %w", ErrInvalidMetric, errUnknownType)
		}
	}

	return s.storage.SetBatch(ctx, metrics)
}
