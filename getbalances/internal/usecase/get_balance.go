package usecase

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"draftea-challenge/getbalances/internal/domain"
)

//go:generate mockgen -source=get_balance.go -destination=mocks/mock_repositories.go -package=mocks

type WalletRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error)
}

type GetBalance struct {
	walletRepo WalletRepository
}

func NewGetBalance(walletRepo WalletRepository) *GetBalance {
	return &GetBalance{walletRepo: walletRepo}
}

func (uc *GetBalance) Execute(ctx context.Context, userID string) (*domain.Wallet, error) {
	ctx, span := otel.Tracer("getbalances").Start(ctx, "UseCase.GetBalance")
	defer span.End()

	slog.InfoContext(ctx, "getting wallet balance", "user_id", userID)
	span.SetAttributes(attribute.String("user_id", userID))

	wallet, err := uc.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to get wallet")
		slog.ErrorContext(ctx, "failed to get wallet", "user_id", userID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("wallet_id", wallet.ID))
	return wallet, nil
}
