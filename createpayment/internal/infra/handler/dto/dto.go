package dto

type CreatePaymentRequest struct {
	Amount float64 `json:"amount"`
}

type CreatePaymentResponse struct {
	Status string `json:"status"`
}
