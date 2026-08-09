package agent

import (
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
	pollCount  int64
	getTicker  *time.Ticker
	sendTicker *time.Ticker
	stopChan   chan bool
	url        string
	httpClient http.Client
}

func NewAgent(storage *repository.MemStorage, url string, httpClient http.Client) *Agent {
	return &Agent{
		storage:    storage,
		pollCount:  0,
		stopChan:   make(chan bool),
		url:        url,
		httpClient: httpClient,
	}
}

func (a *Agent) Start(pollInterval time.Duration, reportInterval time.Duration) {
	a.getTicker = time.NewTicker(pollInterval)
	a.sendTicker = time.NewTicker(reportInterval)

	go func() {
		for {
			select {
			case <-a.getTicker.C:
				a.collectMetrics()
			case <-a.sendTicker.C:
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
					}

					res, err := a.httpClient.Post(fmt.Sprintf("%s/update/%s/%s/%s", a.url, mType, name, mValue), "text/plain", strings.NewReader(""))

					if err != nil {
						log.Printf("Request error: \n%s\n", err.Error())
						a.Stop()
						return
					}
					defer res.Body.Close()
				}
				a.storage.Clear()

			case <-a.stopChan:
				a.getTicker.Stop()
				a.sendTicker.Stop()
				return
			}
		}
	}()
}

func (a *Agent) Stop() {
	close(a.stopChan)
}

func (a *Agent) collectMetrics() {
	a.pollCount++

	a.collectRuntimeMetrics()
	a.collectCustomMetrics()

	log.Printf("Metrics are updated pollCount: %d", a.pollCount)
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
