package domain

import "time"

type TransactionType string

const (
	TransactionTypeCredit  TransactionType = "CREDIT"
	TransactionTypeDebit   TransactionType = "DEBIT"
	TransactionTypeRefund  TransactionType = "REFUND"
	TransactionTypeReserve TransactionType = "RESERVE"
	TransactionTypeRelease TransactionType = "RELEASE"
)

type Transaction struct {
	ID          string
	WalletID    string
	Type        TransactionType
	Amount      float64
	ReferenceID *string
	Description string
	CreatedAt   time.Time
}
