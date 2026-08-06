package service

import (
	"errors"
	"strconv"

	models "github.com/kheguy/collector/internal/model"
)

var (
	parseFloatError = errors.New("can't parse gauge string to float64")
	parseIntError   = errors.New("can't parse counter string to int")
	uknownTypeError = errors.New("unknown metric type")
)

type Storage interface {
	Get(name string) interface{}
	GetAll() map[string]interface{}
	Set(name string, mType string, value interface{}) interface{}
}

type MetricsService struct {
	storage Storage
}

func MakeNewMetricsService(s Storage) *MetricsService {
	return &MetricsService{
		storage: s,
	}
}

func (s *MetricsService) GetMetric(name string) string {
	value := s.storage.Get(name)

	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return ""
	}
}

func (s *MetricsService) GetAllMetrics() map[string]interface{} {
	return s.storage.GetAll()
}

func (s *MetricsService) UpdateMetrics(name string, typeOfValue string, value interface{}) error {

	var parsedValue interface{}

	switch typeOfValue {
	case models.Gauge:
		if str, ok := value.(string); ok {
			if val, err := strconv.ParseFloat(str, 64); err == nil {
				parsedValue = val
			} else {
				return parseFloatError
			}
		}

	case models.Counter:
		if str, ok := value.(string); ok {
			if val, err := strconv.Atoi(str); err == nil {
				parsedValue = val
			} else {
				return parseIntError
			}
		}
	default:
		return uknownTypeError
	}

	s.storage.Set(name, typeOfValue, parsedValue)

	return nil
}
