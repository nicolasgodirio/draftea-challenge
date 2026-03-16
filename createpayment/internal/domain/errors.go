package domain

import "errors"

var (
	ErrWalletNotFound     = errors.New("wallet not found")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInvalidRequestBody = errors.New("invalid request body")
	ErrUnauthorized       = errors.New("unauthorized")
)
