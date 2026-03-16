package handler

import (
	"errors"
	"net/http"

	"draftea-challenge/createpayment/internal/domain"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

var errorStatusMap = map[error]int{
	domain.ErrInvalidAmount:      http.StatusBadRequest,
	domain.ErrInvalidRequestBody: http.StatusBadRequest,
	domain.ErrWalletNotFound:     http.StatusNotFound,
	domain.ErrInsufficientFunds:  http.StatusUnprocessableEntity,
	domain.ErrUnauthorized:       http.StatusUnauthorized,
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
