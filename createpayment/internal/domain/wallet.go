package domain

import "time"

type Wallet struct {
	ID        string
	UserID    string
	Balance   float64
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
