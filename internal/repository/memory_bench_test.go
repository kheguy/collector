package repository

import (
	"context"
	"strconv"
	"testing"

	models "github.com/kheguy/collector/internal/model"
)

func BenchmarkMemoryStorage_SetBatch(b *testing.B) {
	for _, size := range []int{32, 1024} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			ctx := context.Background()
			storage := NewMemoryStorage()
			metrics := memoryBenchmarkMetrics(size)
			if err := storage.SetBatch(ctx, metrics); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			for b.Loop() {
				if err := storage.SetBatch(ctx, metrics); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkMemoryStorage_GetAll(b *testing.B) {
	for _, size := range []int{32, 1024} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			ctx := context.Background()
			storage := NewMemoryStorage()
			if err := storage.SetBatch(ctx, memoryBenchmarkMetrics(size)); err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			for b.Loop() {
				if _, err := storage.GetAll(ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func memoryBenchmarkMetrics(size int) []models.Metrics {
	metrics := make([]models.Metrics, size)
	for i := range metrics {
		metrics[i].ID = "metric_" + strconv.Itoa(i)
		if i%2 == 0 {
			value := float64(i) + 0.5
			metrics[i].MType = models.Gauge
			metrics[i].Value = &value
		} else {
			delta := int64(1)
			metrics[i].MType = models.Counter
			metrics[i].Delta = &delta
		}
	}
	return metrics
}
