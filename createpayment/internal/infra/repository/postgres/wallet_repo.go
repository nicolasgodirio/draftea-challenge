package postgres

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"

	"draftea-challenge/createpayment/internal/domain"
)

type WalletModel struct {
	ID        string    `gorm:"primaryKey;column:id"`
	UserID    string    `gorm:"column:user_id"`
	Balance   float64   `gorm:"column:balance"`
	Currency  string    `gorm:"column:currency"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (WalletModel) TableName() string {
	return "wallets"
}

type WalletRepo struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) *WalletRepo {
	return &WalletRepo{db: db}
}

func (r *WalletRepo) GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error) {
	ctx, span := otel.Tracer("createpayment").Start(ctx, "WalletRepo.GetByUserID")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	var model WalletModel
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&model)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			span.SetStatus(codes.Error, "wallet not found")
			return nil, domain.ErrWalletNotFound
		}
		span.RecordError(result.Error)
		span.SetStatus(codes.Error, "failed to get wallet")
		return nil, result.Error
	}

	span.SetAttributes(attribute.String("wallet_id", model.ID))
	return &domain.Wallet{
		ID:        model.ID,
		UserID:    model.UserID,
		Balance:   model.Balance,
		Currency:  model.Currency,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}, nil
}
