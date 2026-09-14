package handler

import (
	"context"
	"net/http"
)

type Pool interface {
	Ping(context.Context) error
}

type HealthHandler struct {
	pool Pool
}

func MakeNewHealthHandler(p Pool) *HealthHandler {
	return &HealthHandler{
		pool: p,
	}
}

func (h *HealthHandler) PingHandler(res http.ResponseWriter, req *http.Request) {
	err := h.pool.Ping(req.Context())
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	res.Write([]byte("ok"))
}
