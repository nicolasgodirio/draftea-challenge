package usecase

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"draftea-challenge/createpayment/internal/domain"
)

//go:generate mockgen -source=create_payment.go -destination=mocks/mock_repositories.go -package=mocks

type WalletRepository interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Wallet, error)
}

type PaymentRepository interface {
	Create(ctx context.Context, payment *domain.Payment) error
}

type PaymentPublisher interface {
	Publish(ctx context.Context, payment *domain.Payment) error
}

type CreatePaymentInput struct {
	UserID string
	Amount float64
}

type CreatePayment struct {
	walletRepo  WalletRepository
	paymentRepo PaymentRepository
	publisher   PaymentPublisher
}

func NewCreatePayment(walletRepo WalletRepository, paymentRepo PaymentRepository, publisher PaymentPublisher) *CreatePayment {
	return &CreatePayment{walletRepo: walletRepo, paymentRepo: paymentRepo, publisher: publisher}
}

func (uc *CreatePayment) Execute(ctx context.Context, input CreatePaymentInput) (*domain.Payment, error) {
	ctx, span := otel.Tracer("createpayment").Start(ctx, "UseCase.CreatePayment")
	defer span.End()

	slog.InfoContext(ctx, "creating payment", "user_id", input.UserID, "amount", input.Amount)

	span.SetAttributes(
		attribute.String("user_id", input.UserID),
		attribute.Float64("amount", input.Amount),
	)

	wallet, err := uc.walletRepo.GetByUserID(ctx, input.UserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "wallet not found")
		slog.ErrorContext(ctx, "wallet not found", "user_id", input.UserID, "error", err)
		return nil, err
	}

	if wallet.Balance < input.Amount {
		span.SetStatus(codes.Error, "insufficient funds")
		slog.WarnContext(ctx, "insufficient funds", "user_id", input.UserID, "balance", wallet.Balance, "amount", input.Amount)
		return nil, domain.ErrInsufficientFunds
	}

	payment := &domain.Payment{
		UserID:   input.UserID,
		WalletID: wallet.ID,
		Amount:   input.Amount,
		Currency: wallet.Currency,
		Status:   domain.PaymentStatusPending,
	}

	if err := uc.paymentRepo.Create(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create payment")
		slog.ErrorContext(ctx, "failed to create payment", "error", err)
		return nil, err
	}

	if err := uc.publisher.Publish(ctx, payment); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to publish payment event")
		slog.ErrorContext(ctx, "failed to publish payment event", "payment_id", payment.ID, "error", err)
		return nil, err
	}

	span.SetAttributes(attribute.String("payment_id", payment.ID))
	slog.InfoContext(ctx, "payment created and published", "payment_id", payment.ID, "status", payment.Status)
	return payment, nil
}
