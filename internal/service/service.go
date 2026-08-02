package service

import (
	"errors"
	"strconv"

	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
)

var (
	parseFloatError = errors.New("can't parse gauge string to float64")
	parseIntError   = errors.New("can't parse counter string to int")
	uknownTypeError = errors.New("unknown metric type")
)

type MetricsService struct {
	Storage *repository.MemStorage
}

func MakeNewMetricsService(s *repository.MemStorage) *MetricsService {
	return &MetricsService{
		Storage: s,
	}
}

func (s *MetricsService) GetMetric(name string) string {
	value := s.Storage.Get(name)

	if value == nil {
		return ""
	}

	if name == "PollCount" {
		return strconv.Itoa(value.(int))
	} else {
		return strconv.FormatFloat(value.(float64), 'f', -1, 64)
	}
}

func (s *MetricsService) GetAllMetrics() map[string]interface{} {
	return s.Storage.GetAll()
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

	s.Storage.Set(name, typeOfValue, parsedValue)

	return nil
}
