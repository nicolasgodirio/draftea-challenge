package handler

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /balances", h.Handle)
	return AuthMiddleware(mux)
}
