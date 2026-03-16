package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"gorm.io/gorm"

	"draftea-challenge/createpayment/internal/domain"
)

type PaymentModel struct {
	ID             string               `gorm:"primaryKey;column:id"`
	UserID         string               `gorm:"column:user_id"`
	WalletID       string               `gorm:"column:wallet_id"`
	Amount         float64              `gorm:"column:amount"`
	Currency       string               `gorm:"column:currency"`
	Status         domain.PaymentStatus `gorm:"column:status"`
	IdempotencyKey string               `gorm:"column:idempotency_key"`
	CreatedAt      time.Time            `gorm:"column:created_at"`
	UpdatedAt      time.Time            `gorm:"column:updated_at"`
}

func (PaymentModel) TableName() string {
	return "payments"
}

type PaymentRepo struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

func (r *PaymentRepo) Create(ctx context.Context, payment *domain.Payment) error {
	ctx, span := otel.Tracer("createpayment").Start(ctx, "PaymentRepo.Create")
	defer span.End()

	model := PaymentModel{
		ID:             uuid.New().String(),
		UserID:         payment.UserID,
		WalletID:       payment.WalletID,
		Amount:         payment.Amount,
		Currency:       payment.Currency,
		Status:         payment.Status,
		IdempotencyKey: uuid.New().String(),
	}

	result := r.db.WithContext(ctx).Create(&model)
	if result.Error != nil {
		span.RecordError(result.Error)
		span.SetStatus(codes.Error, "failed to create payment")
		return result.Error
	}

	payment.ID = model.ID
	payment.CreatedAt = model.CreatedAt
	payment.UpdatedAt = model.UpdatedAt

	span.SetAttributes(attribute.String("payment_id", payment.ID))
	return nil
}
