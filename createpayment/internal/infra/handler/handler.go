package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"draftea-challenge/createpayment/internal/infra/handler/dto"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"draftea-challenge/createpayment/internal/domain"
	"draftea-challenge/createpayment/internal/usecase"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_usecase.go -package=mocks

type PaymentCreator interface {
	Execute(ctx context.Context, input usecase.CreatePaymentInput) (*domain.Payment, error)
}

type Handler struct {
	createPayment PaymentCreator
}

func New(uc PaymentCreator) *Handler {
	return &Handler{createPayment: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("createpayment").Start(r.Context(), "Handler.Handle")
	defer span.End()

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		span.SetStatus(codes.Error, "unauthorized")
		handleError(w, domain.ErrUnauthorized)
		return
	}

	span.SetAttributes(attribute.String("user_id", userID))

	var req dto.CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.SetStatus(codes.Error, "invalid request body")
		handleError(w, domain.ErrInvalidRequestBody)
		return
	}

	span.SetAttributes(attribute.Float64("amount", req.Amount))

	if req.Amount <= 0 {
		span.SetStatus(codes.Error, "invalid amount")
		handleError(w, domain.ErrInvalidAmount)
		return
	}

	payment, err := h.createPayment.Execute(ctx, usecase.CreatePaymentInput{
		UserID: userID,
		Amount: req.Amount,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		handleError(w, err)
		return
	}

	span.SetAttributes(attribute.String("payment_id", payment.ID))
	writeJSON(w, http.StatusAccepted, dto.CreatePaymentResponse{
		Status: string(payment.Status),
	})
}
