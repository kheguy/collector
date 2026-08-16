package agent

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strconv"
	"strings"
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
		var mType string
		var mValue string

		switch v := value.(type) {
		case int:
			mType = models.Counter
			mValue = strconv.Itoa(v)
		case float64:
			mType = models.Gauge
			mValue = strconv.FormatFloat(v, 'f', -1, 64)
		default:
			continue
		}

		// Вроде вот так через конитекст еще и запросы можно зацепить
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			fmt.Sprintf("%s/update/%s/%s/%s", a.url, mType, name, mValue),
			strings.NewReader(""),
		)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "text/plain")

		res, err := a.httpClient.Do(req)
		if err != nil {
			return err
		}
		res.Body.Close()
	}

	a.storage.Clear()
	return nil
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
