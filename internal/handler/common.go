package handler

import (
	"net/http"
)

type CommonHandler struct{}

func MakeNewCommonHandler() *CommonHandler {
	return &CommonHandler{}
}

func (h *CommonHandler) NotFoundHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusNotFound)
	res.Write([]byte(`Not found`))
}
