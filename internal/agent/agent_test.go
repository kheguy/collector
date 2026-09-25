package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
)

const testAgentURL = "http://collector.test"

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPClient() http.Client {
	return http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       http.NoBody,
				Request:    req,
			}, nil
		}),
	}
}

func TestNewAgent(t *testing.T) {
	storage := repository.NewMemoryStorage()

	agent := NewAgent(storage, testAgentURL, testHTTPClient())

	assert.NotNil(t, agent)
	assert.NotNil(t, agent.url)
}

func TestAgent_CollectRuntimeMetrics(t *testing.T) {
	storage := repository.NewMemoryStorage()
	ctx := context.Background()

	agent := NewAgent(storage, testAgentURL, testHTTPClient())

	err := agent.collectRuntimeMetrics(ctx)
	assert.NoError(t, err)

	metrics, err := storage.GetAll(ctx)
	assert.NoError(t, err)
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
	storage := repository.NewMemoryStorage()
	ctx := context.Background()

	err := storage.Set(ctx, "PollCount", models.Counter, 5)
	assert.NoError(t, err)

	agent := NewAgent(storage, testAgentURL, testHTTPClient())

	err = agent.collectCustomMetrics(ctx)
	assert.NoError(t, err)

	pollCount, err := storage.Get(ctx, "PollCount")
	assert.NoError(t, err)
	assert.NotNil(t, pollCount)
	assert.Equal(t, 6, pollCount)

	randomValue, err := storage.Get(ctx, "RandomValue")
	assert.NoError(t, err)
	assert.NotNil(t, randomValue)
}

func TestAgent_CollectCustomMetrics_WithNoExistingPollCount(t *testing.T) {
	storage := repository.NewMemoryStorage()
	ctx := context.Background()

	agent := NewAgent(storage, testAgentURL, testHTTPClient())

	// Не устанавливаем PollCount заранее
	err := agent.collectCustomMetrics(ctx)
	assert.NoError(t, err)

	pollCount, err := storage.Get(ctx, "PollCount")
	assert.NoError(t, err)
	assert.NotNil(t, pollCount)
	assert.Equal(t, 1, pollCount.(int))
}

func TestAgent_CollectMetrics(t *testing.T) {
	storage := repository.NewMemoryStorage()
	ctx := context.Background()

	agent := NewAgent(storage, testAgentURL, testHTTPClient())

	err := agent.collectMetrics(ctx)
	assert.NoError(t, err)

	metrics, err := storage.GetAll(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)

	pollCount, err := storage.Get(ctx, "PollCount")
	assert.NoError(t, err)
	assert.NotNil(t, pollCount)
	assert.Equal(t, 1, pollCount.(int))

	randomValue, err := storage.Get(ctx, "RandomValue")
	assert.NoError(t, err)
	assert.NotNil(t, randomValue)

	// Пару метрик проверяем
	assert.Contains(t, metrics, "Alloc")
	assert.Contains(t, metrics, "HeapAlloc")
}

func TestAgent_RunStopsAfterContextCancellation(t *testing.T) {
	storage := repository.NewMemoryStorage()

	agent := NewAgent(storage, testAgentURL, testHTTPClient())

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
	storage := repository.NewMemoryStorage()

	agent := NewAgent(storage, testAgentURL, testHTTPClient())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		agent.Run(ctx, 2*time.Millisecond, 10*time.Millisecond)
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()
	<-done

	metrics, err := storage.GetAll(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, metrics)
	assert.Contains(t, metrics, "PollCount")
	assert.Greater(t, metrics["PollCount"].(int), 0)
	assert.Contains(t, metrics, "RandomValue")
	assert.Contains(t, metrics, "Alloc")
}

func TestAgent_SendMetricsBatchSuccess(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemoryStorage()
	require.NoError(t, storage.Set(ctx, "requests", models.Counter, 3))
	require.NoError(t, storage.Set(ctx, "temperature", models.Gauge, 7.5))

	requests := 0
	var batch []models.Metrics
	client := http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/updates/", req.URL.Path)
		assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
		reader, err := gzip.NewReader(req.Body)
		require.NoError(t, err)
		defer reader.Close()
		require.NoError(t, json.NewDecoder(reader).Decode(&batch))
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: http.NoBody}, nil
	})}

	agent := NewAgent(storage, testAgentURL, client)
	require.NoError(t, agent.sendMetrics(ctx))
	assert.Equal(t, 1, requests)

	delta, value := int64(3), 7.5
	assert.ElementsMatch(t, []models.Metrics{
		{ID: "requests", MType: models.Counter, Delta: &delta},
		{ID: "temperature", MType: models.Gauge, Value: &value},
	}, batch)

	remaining, err := storage.GetAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestAgent_SendMetricsEmptyBatch(t *testing.T) {
	requests := 0
	client := http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})}

	agent := NewAgent(repository.NewMemoryStorage(), testAgentURL, client)
	require.NoError(t, agent.sendMetrics(context.Background()))
	assert.Zero(t, requests)
}

func TestAgent_SendMetricsNon2xxKeepsMetrics(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemoryStorage()
	require.NoError(t, storage.Set(ctx, "temperature", models.Gauge, 7.5))

	client := http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Status:     "500 Internal Server Error",
			Body:       http.NoBody,
		}, nil
	})}

	agent := NewAgent(storage, testAgentURL, client)
	assert.Error(t, agent.sendMetrics(ctx))

	remaining, err := storage.Get(ctx, "temperature")
	require.NoError(t, err)
	assert.Equal(t, 7.5, remaining)
}
