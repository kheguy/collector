package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kheguy/collector/internal/mocks"
	models "github.com/kheguy/collector/internal/model"
	"github.com/kheguy/collector/internal/repository"
	"github.com/kheguy/collector/internal/signature"
)

const testAgentURL = "http://collector.test"

func testHTTPClient(t *testing.T) *mocks.MockHTTPClient {
	return mocks.NewMockHTTPClient(t)
}

func useTestSystemMetrics(t *testing.T, agent *Agent) {
	system := mocks.NewMockSystemMetrics(t)
	system.EXPECT().VirtualMemory(mock.Anything).Return(uint64(1024), uint64(512), nil).Maybe()
	system.EXPECT().CPUPercent(mock.Anything).Return([]float64{10}, nil).Maybe()
	agent.system = system
}

func TestNewAgent(t *testing.T) {
	storage := repository.NewMemoryStorage()

	agent := NewAgent(storage, testAgentURL, testHTTPClient(t), "")

	assert.NotNil(t, agent)
	assert.NotNil(t, agent.url)
}

func TestAgent_CollectRuntimeMetrics(t *testing.T) {
	storage := repository.NewMemoryStorage()
	ctx := context.Background()

	agent := NewAgent(storage, testAgentURL, testHTTPClient(t), "")

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

	agent := NewAgent(storage, testAgentURL, testHTTPClient(t), "")

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

	agent := NewAgent(storage, testAgentURL, testHTTPClient(t), "")

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

	agent := NewAgent(storage, testAgentURL, testHTTPClient(t), "")

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

	agent := NewAgent(storage, testAgentURL, testHTTPClient(t), "")
	useTestSystemMetrics(t, agent)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		agent.Run(ctx, 2*time.Millisecond, 10*time.Millisecond, 1)
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

	agent := NewAgent(storage, testAgentURL, testHTTPClient(t), "")
	useTestSystemMetrics(t, agent)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		agent.Run(ctx, 2*time.Millisecond, 10*time.Millisecond, 1)
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

	var batch []models.Metrics
	client := mocks.NewMockHTTPClient(t)
	client.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodPost, req.Method)
		assert.Equal(t, "/updates/", req.URL.Path)
		assert.Equal(t, "gzip", req.Header.Get("Content-Encoding"))
		reader, err := gzip.NewReader(req.Body)
		require.NoError(t, err)
		defer reader.Close()
		require.NoError(t, json.NewDecoder(reader).Decode(&batch))
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: http.NoBody}, nil
	}).Once()

	agent := NewAgent(storage, testAgentURL, client, "")
	require.NoError(t, agent.sendMetrics(ctx))

	delta, value := int64(3), 7.5
	assert.ElementsMatch(t, []models.Metrics{
		{ID: "requests", MType: models.Counter, Delta: &delta},
		{ID: "temperature", MType: models.Gauge, Value: &value},
	}, batch)

	remaining, err := storage.GetAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestAgent_SendMetricsKeepsMetricsCollectedDuringRequest(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemoryStorage()
	require.NoError(t, storage.Set(ctx, "old", models.Gauge, 1.0))

	requestStarted := make(chan struct{})
	releaseRequest := make(chan struct{})
	client := mocks.NewMockHTTPClient(t)
	client.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
		close(requestStarted)
		<-releaseRequest
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: http.NoBody}, nil
	}).Once()

	agent := NewAgent(storage, testAgentURL, client, "")
	sendDone := make(chan error, 1)
	go func() {
		sendDone <- agent.sendMetrics(ctx)
	}()

	<-requestStarted
	require.NoError(t, storage.Set(ctx, "new", models.Gauge, 2.0))
	close(releaseRequest)
	require.NoError(t, <-sendDone)

	remaining, err := storage.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"new": 2.0}, remaining)
}

func TestAgent_SendMetricsEmptyBatch(t *testing.T) {
	client := mocks.NewMockHTTPClient(t)

	agent := NewAgent(repository.NewMemoryStorage(), testAgentURL, client, "")
	require.NoError(t, agent.sendMetrics(context.Background()))
}

func TestAgent_SendMetricsSignsRequest(t *testing.T) {
	const key = "secret"

	ctx := context.Background()
	storage := repository.NewMemoryStorage()
	require.NoError(t, storage.Set(ctx, "temperature", models.Gauge, 7.5))

	client := mocks.NewMockHTTPClient(t)
	client.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, signature.Calculate(body, key), req.Header.Get(signature.Header))
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: http.NoBody}, nil
	}).Once()

	agent := NewAgent(storage, testAgentURL, client, key)
	require.NoError(t, agent.sendMetrics(ctx))
}

func TestAgent_SendMetricsNon2xxKeepsMetrics(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemoryStorage()
	require.NoError(t, storage.Set(ctx, "temperature", models.Gauge, 7.5))

	client := mocks.NewMockHTTPClient(t)
	client.EXPECT().Do(mock.Anything).Return(
		&http.Response{
			StatusCode: http.StatusInternalServerError,
			Status:     "500 Internal Server Error",
			Body:       http.NoBody,
		}, nil,
	).Once()

	agent := NewAgent(storage, testAgentURL, client, "")
	assert.Error(t, agent.sendMetrics(ctx))

	remaining, err := storage.Get(ctx, "temperature")
	require.NoError(t, err)
	assert.Equal(t, 7.5, remaining)
}

func TestAgent_CollectGopsutilMetrics(t *testing.T) {
	ctx := context.Background()
	storage := repository.NewMemoryStorage()
	agent := NewAgent(storage, testAgentURL, testHTTPClient(t), "")
	system := mocks.NewMockSystemMetrics(t)
	system.EXPECT().VirtualMemory(ctx).Return(uint64(2048), uint64(1024), nil).Once()
	system.EXPECT().CPUPercent(ctx).Return([]float64{12.5, 25}, nil).Once()
	agent.system = system

	require.NoError(t, agent.collectGopsutilMetrics(ctx))

	metrics, err := storage.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"TotalMemory":     float64(2048),
		"FreeMemory":      float64(1024),
		"CPUutilization1": 12.5,
		"CPUutilization2": 25.0,
	}, metrics)
}

func TestAgent_RunLimitsConcurrentRequests(t *testing.T) {
	const rateLimit = 2

	storage := repository.NewMemoryStorage()
	require.NoError(t, storage.Set(context.Background(), "temperature", models.Gauge, 7.5))

	var active atomic.Int32
	var maximum atomic.Int32
	client := mocks.NewMockHTTPClient(t)
	client.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
		current := active.Add(1)
		defer active.Add(-1)

		for {
			previous := maximum.Load()
			if current <= previous || maximum.CompareAndSwap(previous, current) {
				break
			}
		}

		<-req.Context().Done()
		return nil, req.Context().Err()
	})

	agent := NewAgent(storage, testAgentURL, client, "")
	useTestSystemMetrics(t, agent)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		agent.Run(ctx, time.Millisecond, time.Millisecond, rateLimit)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return maximum.Load() == rateLimit
	}, time.Second, time.Millisecond)
	assert.LessOrEqual(t, maximum.Load(), int32(rateLimit))

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run should stop after context cancellation")
	}
}
