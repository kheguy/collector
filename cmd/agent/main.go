package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/kheguy/collector/internal/config"
	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
)

type Agent struct {
	storage    *repository.MemStorage
	pollCount  int64
	getTicker  *time.Ticker
	sendTicker *time.Ticker
	stopChan   chan bool
	config     config.Config
}

func NewAgent(storage *repository.MemStorage) *Agent {
	return &Agent{
		storage:   storage,
		pollCount: 0,
		stopChan:  make(chan bool),
		config:    config.Load(),
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
					// Не вижу смысла заводить type в store под единственный счетчик
					var mType string
					var mValue string
					if name == "PollCount" {
						mType = models.Counter
						mValue = strconv.Itoa(value.(int))
					} else {
						mType = models.Gauge
						mValue = strconv.FormatFloat(value.(float64), 'f', -1, 64)
					}

					_, err := http.Post(fmt.Sprintf("http://%s:%d/update/%s/%s/%s", a.config.Host, a.config.Port, mType, name, mValue), "plain/text", strings.NewReader(""))

					if err != nil {
						fmt.Printf("Request error: \n%s\n", err.Error())
						a.Stop()
					}
				}

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

	fmt.Printf("Metrics are updated pollCount: %d, time is: %s\n",
		a.pollCount, time.Now().Format("15:04:05"))
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
	pollCount := a.storage.Get("PollCount")
	if pollCount == nil {
		pollCount = 0
	}

	a.storage.Set("PollCount", models.Counter, pollCount.(int)+1)

	randomValue := rand.Float64() * 100
	a.storage.Set("RandomValue", models.Gauge, randomValue)
}

func main() {
	storage := repository.MakeNewMemoryStorage()

	agent := NewAgent(storage)

	agent.Start(2*time.Second, 5*time.Second)

	time.Sleep(100000000 * time.Second)
}
