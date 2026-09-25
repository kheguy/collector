package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotFoundHandler(t *testing.T) {
	handler := MakeNewCommonHandler()

	req := httptest.NewRequest("GET", "/any", nil)
	w := httptest.NewRecorder()

	handler.NotFoundHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "Not found", w.Body.String())
}
