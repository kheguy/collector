package handler

import (
	"net/http"
)

// CommonHandler содержит общие HTTP обработчики
type CommonHandler struct{}

// MakeNewCommonHandler создаёт набор общих HTTP обработчиков
func MakeNewCommonHandler() *CommonHandler {
	return &CommonHandler{}
}

// NotFoundHandler возвращает статус 404 и сообщение Not found
func (h *CommonHandler) NotFoundHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusNotFound)
	res.Write([]byte(`Not found`))
}
