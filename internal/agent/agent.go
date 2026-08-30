package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"time"

	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
)

type Agent struct {
	storage    *repository.MemStorage
	url        string
	httpClient http.Client
}

func NewAgent(storage *repository.MemStorage, url string, httpClient http.Client) *Agent {
	return &Agent{
		storage:    storage,
		url:        url,
		httpClient: httpClient,
	}
}

func (a *Agent) Run(ctx context.Context, pollInterval time.Duration, reportInterval time.Duration) {
	getTicker := time.NewTicker(pollInterval)
	sendTicker := time.NewTicker(reportInterval)
	defer getTicker.Stop()
	defer sendTicker.Stop()

	for {
		select {
		case <-getTicker.C:
			a.collectMetrics()
		case <-sendTicker.C:
			if err := a.sendMetrics(ctx); err != nil {
				log.Printf("Request error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) sendMetrics(ctx context.Context) error {
	for name, value := range a.storage.GetAll() {
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

		body, err := json.Marshal(metric)
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
			fmt.Sprintf("%s/update/", a.url),
			bytes.NewReader(compressedBody),
		)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Encoding", "gzip")

		res, err := a.httpClient.Do(req)
		if err != nil {
			return err
		}

		if _, err := io.Copy(io.Discard, res.Body); err != nil {
			res.Body.Close()
			return err
		}
		res.Body.Close()
		if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("server returned status %s", res.Status)
		}
	}

	a.storage.Clear()
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

func (a *Agent) collectMetrics() {
	a.collectRuntimeMetrics()
	a.collectCustomMetrics()

	log.Printf("Metrics are updated pollCount: %v", a.storage.Get("PollCount"))
}

func (a *Agent) collectRuntimeMetrics() {
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
		a.storage.Set(name, models.Gauge, value)
	}
}

func (a *Agent) collectCustomMetrics() {
	a.storage.Set("PollCount", models.Counter, 1)

	randomValue := rand.Float64() * 100
	a.storage.Set("RandomValue", models.Gauge, randomValue)
}
