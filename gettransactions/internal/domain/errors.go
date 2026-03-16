package domain

import "errors"

var (
	ErrWalletNotFound = errors.New("wallet not found")
	ErrUnauthorized   = errors.New("unauthorized")
)
