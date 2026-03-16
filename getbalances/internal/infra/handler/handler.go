package handler

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"draftea-challenge/getbalances/internal/domain"
	"draftea-challenge/getbalances/internal/infra/handler/dto"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_usecase.go -package=mocks

type BalanceGetter interface {
	Execute(ctx context.Context, userID string) (*domain.Wallet, error)
}

type Handler struct {
	getBalance BalanceGetter
}

func New(uc BalanceGetter) *Handler {
	return &Handler{getBalance: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("getbalances").Start(r.Context(), "Handler.Handle")
	defer span.End()

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		span.SetStatus(codes.Error, "unauthorized")
		handleError(w, domain.ErrUnauthorized)
		return
	}

	span.SetAttributes(attribute.String("user_id", userID))

	wallet, err := h.getBalance.Execute(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		handleError(w, err)
		return
	}

	span.SetAttributes(attribute.String("wallet_id", wallet.ID))
	writeJSON(w, http.StatusOK, dto.BalanceResponse{
		UserID:   wallet.UserID,
		Balance:  wallet.Balance,
		Currency: wallet.Currency,
	})
}
