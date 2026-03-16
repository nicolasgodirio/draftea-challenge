package handler

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /transactions", h.Handle)
	return AuthMiddleware(mux)
}
