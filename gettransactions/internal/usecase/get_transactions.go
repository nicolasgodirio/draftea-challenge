package usecase

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"draftea-challenge/gettransactions/internal/domain"
)

//go:generate mockgen -source=get_transactions.go -destination=mocks/mock_repositories.go -package=mocks

type WalletRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error)
	GetTransactionsByUserID(ctx context.Context, userID string) ([]domain.Transaction, error)
}

type GetTransactions struct {
	walletRepo WalletRepository
}

func NewGetTransactions(walletRepo WalletRepository) *GetTransactions {
	return &GetTransactions{walletRepo: walletRepo}
}

func (uc *GetTransactions) Execute(ctx context.Context, userID string) ([]domain.Transaction, error) {
	ctx, span := otel.Tracer("gettransactions").Start(ctx, "UseCase.GetTransactions")
	defer span.End()

	slog.InfoContext(ctx, "getting transactions", "user_id", userID)
	span.SetAttributes(attribute.String("user_id", userID))

	_, err := uc.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "wallet not found")
		slog.ErrorContext(ctx, "wallet not found for user", "user_id", userID, "error", err)
		return nil, err
	}

	txns, err := uc.walletRepo.GetTransactionsByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get transactions")
		slog.ErrorContext(ctx, "failed to get transactions", "user_id", userID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.Int("transaction_count", len(txns)))
	return txns, nil
}
