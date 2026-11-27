package agent

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"runtime/pprof"
	"testing"

	"github.com/kheguy/collector/internal/handler"
	"github.com/kheguy/collector/internal/middlewares"
	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/service"
)

var allocProfile = flag.String("allocprofile", "", "Write an allocation profile after BenchmarkAgent_Report")

func benchmarkAgentMetrics() map[string]interface{} {
	return map[string]interface{}{
		"Alloc":           float64(32 << 20),
		"BuckHashSys":     float64(4096),
		"Frees":           float64(20000),
		"GCCPUFraction":   0.003,
		"GCSys":           float64(2 << 20),
		"HeapAlloc":       float64(32 << 20),
		"HeapIdle":        float64(8 << 20),
		"HeapInuse":       float64(34 << 20),
		"HeapObjects":     float64(120000),
		"HeapReleased":    float64(4 << 20),
		"HeapSys":         float64(42 << 20),
		"LastGC":          float64(1795737600000000000),
		"Lookups":         float64(0),
		"MCacheInuse":     float64(9600),
		"MCacheSys":       float64(15600),
		"MSpanInuse":      float64(120000),
		"MSpanSys":        float64(150000),
		"Mallocs":         float64(140000),
		"NextGC":          float64(64 << 20),
		"NumForcedGC":     float64(0),
		"NumGC":           float64(5),
		"OtherSys":        float64(1 << 20),
		"PauseTotalNs":    float64(1000000),
		"StackInuse":      float64(512 << 10),
		"StackSys":        float64(512 << 10),
		"Sys":             float64(48 << 20),
		"TotalAlloc":      float64(64 << 20),
		"PollCount":       100,
		"RandomValue":     42.75,
		"TotalMemory":     float64(8 << 30),
		"FreeMemory":      float64(2 << 30),
		"CPUutilization1": 12.5,
	}
}

type benchmarkReportClient struct {
	handler http.Handler
}

func (c benchmarkReportClient) Do(req *http.Request) (*http.Response, error) {
	defer req.Body.Close()
	response := httptest.NewRecorder()
	c.handler.ServeHTTP(response, req)
	return response.Result(), nil
}

func BenchmarkAgent_Report(b *testing.B) {
	storage := repository.NewMemoryStorage()
	metricsService := service.MakeNewMetricsService(storage)
	metricsHandler := handler.MakeNewMetricsHandler(metricsService, nil)
	client := benchmarkReportClient{
		handler: middlewares.WithGzip(http.HandlerFunc(metricsHandler.JSONBatchUpdateHandler)),
	}
	agent := NewAgent(repository.NewMemoryStorage(), testAgentURL, client, "")
	metrics := benchmarkAgentMetrics()
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if err := agent.sendMetricsBatch(ctx, metrics); err != nil {
			b.Fatal(err)
		}
	}
	if *allocProfile == "" {
		return
	}
	runtime.GC()
	file, err := os.Create(*allocProfile)
	if err != nil {
		b.Fatal(err)
	}
	writeErr := pprof.Lookup("allocs").WriteTo(file, 0)
	closeErr := file.Close()
	if writeErr != nil {
		b.Fatal(writeErr)
	}
	if closeErr != nil {
		b.Fatal(closeErr)
	}
	runtime.KeepAlive(agent)
	runtime.KeepAlive(metrics)
}

func BenchmarkCompress(b *testing.B) {
	metrics := make([]models.Metrics, 32)
	for i := range metrics {
		value := float64(i) * 1024.5
		metrics[i] = models.Metrics{ID: fmt.Sprintf("Gauge%d", i), MType: models.Gauge, Value: &value}
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		if _, err := compress(data); err != nil {
			b.Fatal(err)
		}
	}
}
