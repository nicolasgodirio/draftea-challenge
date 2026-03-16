package dto

import "time"

type TransactionItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Amount      float64   `json:"amount"`
	ReferenceID *string   `json:"reference_id,omitempty"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type TransactionsResponse struct {
	UserID       string            `json:"user_id"`
	Transactions []TransactionItem `json:"transactions"`
}
