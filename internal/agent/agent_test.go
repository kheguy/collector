package agent

import (
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
	assert.Equal(t, int64(0), agent.pollCount)
	assert.NotNil(t, agent.stopChan)
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

	initialPollCount := agent.pollCount

	agent.collectMetrics()

	assert.Equal(t, initialPollCount+1, agent.pollCount)

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

func TestAgent_StartStop(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	agent := NewAgent(storage, server.URL, http.Client{})

	agent.Start(2*time.Millisecond, 10*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	agent.Stop()

	assert.NotNil(t, agent.getTicker)
	assert.NotNil(t, agent.sendTicker)

	select {
	case _, ok := <-agent.stopChan:
		assert.False(t, ok, "stopChan should be closed")
	default:
		t.Error("stopChan should be closed")
	}
}

func TestAgent_Start_CollectsMetrics(t *testing.T) {
	storage := repository.MakeNewMemoryStorage()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	agent := NewAgent(storage, server.URL, http.Client{})

	initialPollCount := agent.pollCount

	agent.Start(2*time.Millisecond, 10*time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	agent.Stop()

	assert.Greater(t, agent.pollCount, initialPollCount)

	metrics := storage.GetAll()
	assert.NotEmpty(t, metrics)
	assert.Contains(t, metrics, "PollCount")
	assert.Contains(t, metrics, "RandomValue")
	assert.Contains(t, metrics, "Alloc")
}
