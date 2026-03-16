package handler

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"draftea-challenge/gettransactions/internal/domain"
	"draftea-challenge/gettransactions/internal/infra/handler/dto"
)

//go:generate mockgen -source=handler.go -destination=mocks/mock_usecase.go -package=mocks

type TransactionsGetter interface {
	Execute(ctx context.Context, userID string) ([]domain.Transaction, error)
}

type Handler struct {
	getTransactions TransactionsGetter
}

func New(uc TransactionsGetter) *Handler {
	return &Handler{getTransactions: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer("gettransactions").Start(r.Context(), "Handler.Handle")
	defer span.End()

	userID, ok := UserIDFromContext(ctx)
	if !ok {
		span.SetStatus(codes.Error, "unauthorized")
		handleError(w, domain.ErrUnauthorized)
		return
	}

	span.SetAttributes(attribute.String("user_id", userID))

	txns, err := h.getTransactions.Execute(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		handleError(w, err)
		return
	}

	items := make([]dto.TransactionItem, 0, len(txns))
	for _, t := range txns {
		items = append(items, dto.TransactionItem{
			ID:          t.ID,
			Type:        string(t.Type),
			Amount:      t.Amount,
			ReferenceID: t.ReferenceID,
			Description: t.Description,
			CreatedAt:   t.CreatedAt,
		})
	}

	span.SetAttributes(attribute.Int("transaction_count", len(items)))
	writeJSON(w, http.StatusOK, dto.TransactionsResponse{
		UserID:       userID,
		Transactions: items,
	})
}
