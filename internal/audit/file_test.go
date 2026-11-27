package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileObserver_AppendsEvents(t *testing.T) {
	for name, ending := range map[string]string{"with newline": "\n", "without newline": ""} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "audit.jsonl")
			first := Event{Timestamp: 1, Metrics: []string{"Alloc"}, IPAddress: "192.168.0.1"}
			second := Event{Timestamp: 2, Metrics: []string{"Frees"}, IPAddress: "192.168.0.2"}
			data, err := json.Marshal(first)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, append(data, ending...), 0600))
			observer, err := NewFileObserver(path)
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, observer.Close()) })

			require.NoError(t, observer.Notify(context.Background(), second))

			data, err = os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, byte('\n'), data[len(data)-1])
			assert.Equal(t, []Event{first, second}, readEvents(t, path))
		})
	}
}

func TestFileObserver_ConcurrentWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	observer, err := NewFileObserver(path)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, observer.Close()) })
	const count = 64
	errors := make(chan error, count)
	var wg sync.WaitGroup
	for i := range count {
		wg.Go(func() {
			errors <- observer.Notify(context.Background(), Event{
				Timestamp: int64(i),
				Metrics:   []string{fmt.Sprintf("metric-%d", i)},
				IPAddress: "192.168.0.42",
			})
		})
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	events := readEvents(t, path)
	require.Len(t, events, count)
	seen := make(map[int64]bool, count)
	for _, event := range events {
		assert.False(t, seen[event.Timestamp])
		seen[event.Timestamp] = true
		assert.Equal(t, []string{fmt.Sprintf("metric-%d", event.Timestamp)}, event.Metrics)
		assert.Equal(t, "192.168.0.42", event.IPAddress)
	}
}

func readEvents(t *testing.T, path string) []Event {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	var events []Event
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event Event
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &event))
		events = append(events, event)
	}
	require.NoError(t, scanner.Err())
	return events
}
