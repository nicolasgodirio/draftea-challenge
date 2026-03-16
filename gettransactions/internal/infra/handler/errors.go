package handler

import (
	"errors"
	"net/http"

	"draftea-challenge/gettransactions/internal/domain"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

var errorStatusMap = map[error]int{
	domain.ErrWalletNotFound: http.StatusNotFound,
	domain.ErrUnauthorized:   http.StatusUnauthorized,
}

func handleError(w http.ResponseWriter, err error) {
	for sentinel, status := range errorStatusMap {
		if errors.Is(err, sentinel) {
			writeJSON(w, status, ErrorResponse{Error: sentinel.Error()})
			return
		}
	}
	writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
}
