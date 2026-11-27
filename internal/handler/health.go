package handler

import (
	"context"
	"log"
	"net/http"
)

// Pinger проверяет доступность хранилища
type Pinger interface {
	Ping(context.Context) error
}

// HealthHandler обрабатывает проверку доступности хранилища
type HealthHandler struct {
	pinger Pinger
}

// MakeNewHealthHandler создаёт обработчик проверки хранилища
func MakeNewHealthHandler(p Pinger) *HealthHandler {
	return &HealthHandler{
		pinger: p,
	}
}

// PingHandler проверяет доступность хранилища
// При успехе возвращает 200 и ok, при ошибке возвращает 500
func (h *HealthHandler) PingHandler(res http.ResponseWriter, req *http.Request) {
	err := h.pinger.Ping(req.Context())
	if err != nil {
		log.Printf("Health check error: %v", err)
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.Write([]byte("ok"))
}
