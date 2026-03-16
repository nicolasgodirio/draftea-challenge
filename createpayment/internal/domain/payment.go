package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusCompleted PaymentStatus = "COMPLETED"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusReversed  PaymentStatus = "REVERSED"
)

type Payment struct {
	ID        string
	UserID    string
	WalletID  string
	Amount    float64
	Currency  string
	Status    PaymentStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
