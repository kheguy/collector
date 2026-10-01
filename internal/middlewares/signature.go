package middlewares

import (
	"bytes"
	"io"
	"net/http"

	"github.com/kheguy/collector/internal/signature"
)

type signatureResponseWriter struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func (w *signatureResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = statusCode
}

func (w *signatureResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.body.Write(data)
}

func (w *signatureResponseWriter) flush(key string) {
	w.Header().Set(signature.Header, signature.Calculate(w.body.Bytes(), key))
	statusCode := w.statusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(statusCode)
	_, _ = w.ResponseWriter.Write(w.body.Bytes())
}

func WithSignature(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if key == "" {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			responseWriter := &signatureResponseWriter{ResponseWriter: w}
			defer responseWriter.flush(key)

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(responseWriter, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))

			receivedSignature := r.Header.Get(signature.Header)
			if receivedSignature != "" && !signature.Valid(body, key, receivedSignature) {
				http.Error(responseWriter, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			next.ServeHTTP(responseWriter, r)
		})
	}
}
