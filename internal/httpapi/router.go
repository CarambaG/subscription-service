package httpapi

import "net/http"

func NewRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/subscriptions", handler.Create)
	mux.HandleFunc("GET /api/v1/subscriptions", handler.List)
	mux.HandleFunc("GET /api/v1/subscriptions/total", handler.CalculateTotal)
	mux.HandleFunc("GET /api/v1/subscriptions/{id}", handler.Get)
	mux.HandleFunc("PUT /api/v1/subscriptions/{id}", handler.Update)
	mux.HandleFunc("DELETE /api/v1/subscriptions/{id}", handler.Delete)
	mux.HandleFunc("GET /health/live", handler.Live)
	mux.HandleFunc("GET /health/ready", handler.Ready)
	return mux
}
