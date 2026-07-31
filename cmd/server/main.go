package main

import (
	"fmt"
	"net/http"
)

var port int = 8080

const (
	gaugeType   = "gauge"
	counterType = "counter"
)

type MemStorage struct {
	Data map[string]interface{}
}

var storage MemStorage

// Тут будем забирать метрику в будущем (logs?)
func (s MemStorage) Get(name string) interface{} {
	return s.Data[name]
}

// Установка метрики
func (s MemStorage) Set(name string, mType string, value interface{}) interface{} {
	switch mType {
	case gaugeType:
		s.Data[name] = value.(float64)
	case counterType:
		s.Data[name] = s.Data[name].(int) + value.(int)
	}

	return s.Data[name]
}

func updateRoute(res http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := req.PathValue("name")

	if name == "" {
		http.Error(res, "Name is required", http.StatusNotFound)
		return
	}

	mType := req.PathValue("type")
	value := req.PathValue("value")

	if (mType != gaugeType && mType != counterType) || len(value) == 0 {
		http.Error(res, "Bad request", http.StatusBadRequest)
		return
	}

	storage.Set(name, mType, value)

	res.WriteHeader(http.StatusOK)
	res.Write([]byte(`OK`))
}

func notFoundRoute(res http.ResponseWriter, req *http.Request) {
	res.Write([]byte(`Not found`))
	res.WriteHeader(http.StatusNotFound)
}

func main() {
	storage = MemStorage{
		Data: make(map[string]interface{}),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(`/update/{type}/{name}/{value}`, updateRoute)
	mux.HandleFunc(`/`, notFoundRoute)

	fmt.Printf("Server started on port %d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux)

	if err != nil {
		panic(err)
	}
}
