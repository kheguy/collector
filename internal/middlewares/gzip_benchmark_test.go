package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkWithGzip(b *testing.B) {
	body := bytes.Repeat([]byte(`{"id":"HeapAlloc","type":"gauge","value":16777216},`), 32)
	body[len(body)-1] = ']'
	body = append([]byte{'['}, body...)
	handler := WithGzip(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(body); err != nil {
			b.Fatal(err)
		}
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	b.ReportAllocs()
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", response.Code)
		}
	}
}
