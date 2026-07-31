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
)

type MetricsService struct {
	Storage *repository.MemStorage
}

func MakeNewMetricsService(s *repository.MemStorage) *MetricsService {
	return &MetricsService{
		Storage: s,
	}
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
	}

	s.Storage.Set(name, typeOfValue, parsedValue)

	return nil
}
