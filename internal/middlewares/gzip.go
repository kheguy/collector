package middlewares

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(io.Discard)
	},
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gzipWriter  *gzip.Writer
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html") {
		w.Header().Set("Content-Encoding", "gzip")
		w.gzipWriter = gzipWriterPool.Get().(*gzip.Writer)
		w.gzipWriter.Reset(w.ResponseWriter)
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.gzipWriter != nil {
		return w.gzipWriter.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *gzipResponseWriter) Close() error {
	if w.gzipWriter == nil {
		return nil
	}
	err := w.gzipWriter.Close()
	w.gzipWriter.Reset(io.Discard)
	gzipWriterPool.Put(w.gzipWriter)
	w.gzipWriter = nil
	return err
}

// WithGzip распаковывает запросы gzip и сжимает ответы JSON и HTML, если клиент поддерживает gzip
func WithGzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			defer reader.Close()
			r.Body = reader
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		compressedWriter := &gzipResponseWriter{ResponseWriter: w}
		next.ServeHTTP(compressedWriter, r)
		_ = compressedWriter.Close()
	})
}
