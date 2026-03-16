package handler

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /payments", h.Handle)
	return AuthMiddleware(mux)
}
