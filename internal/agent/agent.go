package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"sync"
	"time"

	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/signature"
)

type Agent struct {
	storage    *repository.MemoryStorage
	url        string
	key        string
	httpClient HTTPClient
	system     SystemMetrics
}

func NewAgent(storage *repository.MemoryStorage, url string, httpClient HTTPClient, key string) *Agent {
	return &Agent{
		storage:    storage,
		url:        url,
		key:        key,
		httpClient: NewRetryClient(httpClient),
		system:     gopsutilMetrics{},
	}
}

func (a *Agent) Run(ctx context.Context, pollInterval time.Duration, reportInterval time.Duration, rateLimit int) {
	if rateLimit < 1 {
		rateLimit = 1
	}

	jobs := make(chan map[string]interface{}, rateLimit)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		a.collectLoop(ctx, pollInterval, a.collectMetrics)
	}()
	go func() {
		defer wg.Done()
		a.collectLoop(ctx, pollInterval, a.collectGopsutilMetrics)
	}()

	for range rateLimit {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.sendWorker(ctx, jobs)
		}()
	}

	sendTicker := time.NewTicker(reportInterval)
	defer func() {
		sendTicker.Stop()
		close(jobs)
		wg.Wait()
	}()

	for {
		select {
		case <-sendTicker.C:
			metrics, err := a.storage.TakeAll(ctx)
			if err != nil {
				log.Printf("Metrics read error: %v", err)
				continue
			}
			if len(metrics) == 0 {
				continue
			}
			select {
			case jobs <- metrics:
			case <-ctx.Done():
				a.restoreMetrics(metrics)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) collectLoop(
	ctx context.Context,
	interval time.Duration,
	collect func(context.Context) error,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := collect(ctx); err != nil {
				log.Printf("Metrics collection error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) sendWorker(ctx context.Context, jobs <-chan map[string]interface{}) {
	for metrics := range jobs {
		if err := a.sendMetricsBatch(ctx, metrics); err != nil {
			log.Printf("Request error: %v", err)
			a.restoreMetrics(metrics)
		}
	}
}

func (a *Agent) restoreMetrics(metrics map[string]interface{}) {
	if err := a.storage.Restore(context.Background(), metrics); err != nil {
		log.Printf("Metrics restore error: %v", err)
	}
}

func (a *Agent) sendMetricsBatch(ctx context.Context, metrics map[string]interface{}) error {
	batch := make([]models.Metrics, 0, len(metrics))
	for name, value := range metrics {
		metric := models.Metrics{ID: name}

		switch v := value.(type) {
		case int:
			delta := int64(v)
			metric.MType = models.Counter
			metric.Delta = &delta
		case float64:
			metric.MType = models.Gauge
			metric.Value = &v
		default:
			continue
		}

		batch = append(batch, metric)
	}

	if len(batch) == 0 {
		return nil
	}

	body, err := json.Marshal(batch)
	if err != nil {
		return err
	}
	compressedBody, err := compress(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/updates/", a.url),
		bytes.NewReader(compressedBody),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if a.key != "" {
		req.Header.Set(signature.Header, signature.Calculate(compressedBody, a.key))
	}

	res, err := a.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("server returned status %s", res.Status)
	}

	return nil
}

func compress(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (a *Agent) collectMetrics(ctx context.Context) error {
	if err := a.collectRuntimeMetrics(ctx); err != nil {
		return err
	}
	if err := a.collectCustomMetrics(ctx); err != nil {
		return err
	}

	pollCount, err := a.storage.Get(ctx, "PollCount")
	if err != nil {
		return err
	}
	log.Printf("Metrics are updated pollCount: %v", pollCount)
	return nil
}

func (a *Agent) collectRuntimeMetrics(ctx context.Context) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]float64{
		"Alloc":         float64(memStats.Alloc),
		"BuckHashSys":   float64(memStats.BuckHashSys),
		"Frees":         float64(memStats.Frees),
		"GCCPUFraction": float64(memStats.GCCPUFraction),
		"GCSys":         float64(memStats.GCSys),
		"HeapAlloc":     float64(memStats.HeapAlloc),
		"HeapIdle":      float64(memStats.HeapIdle),
		"HeapInuse":     float64(memStats.HeapInuse),
		"HeapObjects":   float64(memStats.HeapObjects),
		"HeapReleased":  float64(memStats.HeapReleased),
		"HeapSys":       float64(memStats.HeapSys),
		"LastGC":        float64(memStats.LastGC),
		"Lookups":       float64(memStats.Lookups),
		"MCacheInuse":   float64(memStats.MCacheInuse),
		"MCacheSys":     float64(memStats.MCacheSys),
		"MSpanInuse":    float64(memStats.MSpanInuse),
		"MSpanSys":      float64(memStats.MSpanSys),
		"Mallocs":       float64(memStats.Mallocs),
		"NextGC":        float64(memStats.NextGC),
		"NumForcedGC":   float64(memStats.NumForcedGC),
		"NumGC":         float64(memStats.NumGC),
		"OtherSys":      float64(memStats.OtherSys),
		"PauseTotalNs":  float64(memStats.PauseTotalNs),
		"StackInuse":    float64(memStats.StackInuse),
		"StackSys":      float64(memStats.StackSys),
		"Sys":           float64(memStats.Sys),
		"TotalAlloc":    float64(memStats.TotalAlloc),
	}

	for name, value := range metrics {
		if err := a.storage.Set(ctx, name, models.Gauge, value); err != nil {
			return err
		}
	}

	return nil
}

func (a *Agent) collectCustomMetrics(ctx context.Context) error {
	if err := a.storage.Set(ctx, "PollCount", models.Counter, 1); err != nil {
		return err
	}

	randomValue := rand.Float64() * 100
	return a.storage.Set(ctx, "RandomValue", models.Gauge, randomValue)
}

func (a *Agent) collectGopsutilMetrics(ctx context.Context) error {
	totalMemory, freeMemory, err := a.system.VirtualMemory(ctx)
	if err != nil {
		return err
	}

	if err := a.storage.Set(ctx, "TotalMemory", models.Gauge, float64(totalMemory)); err != nil {
		return err
	}
	if err := a.storage.Set(ctx, "FreeMemory", models.Gauge, float64(freeMemory)); err != nil {
		return err
	}

	cpuUtilization, err := a.system.CPUPercent(ctx)
	if err != nil {
		return err
	}
	for index, value := range cpuUtilization {
		name := fmt.Sprintf("CPUutilization%d", index+1)
		if err := a.storage.Set(ctx, name, models.Gauge, value); err != nil {
			return err
		}
	}

	return nil
}
