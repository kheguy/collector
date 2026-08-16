package agent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
)

func TestNewAgent(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	agent := NewAgent(storage, server.URL, http.Client{})

	assert.NotNil(t, agent)
	assert.NotNil(t, agent.url)
}

func TestAgent_CollectRuntimeMetrics(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	agent := NewAgent(storage, server.URL, http.Client{})

	agent.collectRuntimeMetrics()

	metrics := storage.GetAll()
	assert.NotEmpty(t, metrics)

	expectedMetrics := []string{
		"Alloc",
		"BuckHashSys",
		"Frees",
		"GCCPUFraction",
		"GCSys",
		"HeapAlloc",
		"HeapIdle",
		"HeapInuse",
		"HeapObjects",
		"HeapReleased",
		"HeapSys",
		"LastGC",
		"Lookups",
		"MCacheInuse",
		"MCacheSys",
		"MSpanInuse",
		"MSpanSys",
		"Mallocs",
		"NextGC",
		"NumForcedGC",
		"NumGC",
		"OtherSys",
		"PauseTotalNs",
		"StackInuse",
		"StackSys",
		"Sys",
		"TotalAlloc",
	}
	for _, metricName := range expectedMetrics {
		assert.Contains(t, metrics, metricName, "Metric %s should exist", metricName)
	}

	for _, value := range metrics {
		_, ok := value.(float64)
		assert.True(t, ok, "All runtime metrics should be float64")
	}
}

func TestAgent_CollectCustomMetrics(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	storage.Set("PollCount", models.Counter, 5)

	agent := NewAgent(storage, server.URL, http.Client{})

	agent.collectCustomMetrics()

	pollCount := storage.Get("PollCount")
	assert.NotNil(t, pollCount)
	assert.Equal(t, 6, pollCount)

	randomValue := storage.Get("RandomValue")
	assert.NotNil(t, randomValue)
}

func TestAgent_CollectCustomMetrics_WithNoExistingPollCount(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	agent := NewAgent(storage, server.URL, http.Client{})

	// Не устанавливаем PollCount заранее
	agent.collectCustomMetrics()

	pollCount := storage.Get("PollCount")
	assert.NotNil(t, pollCount)
	assert.Equal(t, 1, pollCount.(int))
}

func TestAgent_CollectMetrics(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	agent := NewAgent(storage, server.URL, http.Client{})

	agent.collectMetrics()

	metrics := storage.GetAll()
	assert.NotEmpty(t, metrics)

	pollCount := storage.Get("PollCount")
	assert.NotNil(t, pollCount)
	assert.Equal(t, 1, pollCount.(int))

	randomValue := storage.Get("RandomValue")
	assert.NotNil(t, randomValue)

	// Пару метрик проверяем
	assert.Contains(t, metrics, "Alloc")
	assert.Contains(t, metrics, "HeapAlloc")
}

func TestAgent_RunStopsAfterContextCancellation(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	agent := NewAgent(storage, server.URL, http.Client{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		agent.Run(ctx, 2*time.Millisecond, 10*time.Millisecond)
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run should stop after context cancellation")
	}
}

func TestAgent_RunCollectsMetrics(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	agent := NewAgent(storage, server.URL, http.Client{})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		agent.Run(ctx, 2*time.Millisecond, 10*time.Millisecond)
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()
	<-done

	metrics := storage.GetAll()
	assert.NotEmpty(t, metrics)
	assert.Contains(t, metrics, "PollCount")
	assert.Greater(t, metrics["PollCount"].(int), 0)
	assert.Contains(t, metrics, "RandomValue")
	assert.Contains(t, metrics, "Alloc")
}
