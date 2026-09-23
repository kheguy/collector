package handler

import (
	"context"
	"log"
	"net/http"
)

type Pinger interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	pinger Pinger
}

func MakeNewHealthHandler(p Pinger) *HealthHandler {
	return &HealthHandler{
		pinger: p,
	}
}

func (h *HealthHandler) PingHandler(res http.ResponseWriter, req *http.Request) {
	err := h.pinger.Ping(req.Context())
	if err != nil {
		log.Printf("Health check error: %v", err)
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.Write([]byte("ok"))
}
