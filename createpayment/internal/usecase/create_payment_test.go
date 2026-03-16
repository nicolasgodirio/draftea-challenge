package usecase_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"draftea-challenge/createpayment/internal/domain"
	"draftea-challenge/createpayment/internal/usecase"
	"draftea-challenge/createpayment/internal/usecase/mocks"
)

func setupUseCase(t *testing.T) (
	*mocks.MockWalletRepository,
	*mocks.MockPaymentRepository,
	*mocks.MockPaymentPublisher,
	*usecase.CreatePayment,
) {
	ctrl := gomock.NewController(t)
	walletRepo := mocks.NewMockWalletRepository(ctrl)
	paymentRepo := mocks.NewMockPaymentRepository(ctrl)
	publisher := mocks.NewMockPaymentPublisher(ctrl)
	uc := usecase.NewCreatePayment(walletRepo, paymentRepo, publisher)
	return walletRepo, paymentRepo, publisher, uc
}

func TestCreatePayment_Success(t *testing.T) {
	walletRepo, paymentRepo, publisher, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(&domain.Wallet{
			ID:       "wallet-1",
			UserID:   "user-1",
			Balance:  500.00,
			Currency: "ARS",
		}, nil)

	paymentRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p *domain.Payment) error {
			p.ID = "pay-1"
			return nil
		})

	publisher.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(nil)

	payment, err := uc.Execute(context.Background(), usecase.CreatePaymentInput{
		UserID: "user-1",
		Amount: 100.00,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if payment == nil {
		t.Fatal("expected payment, got nil")
	}
	if payment.ID != "pay-1" {
		t.Errorf("expected payment_id pay-1, got %s", payment.ID)
	}
	if payment.Status != domain.PaymentStatusPending {
		t.Errorf("expected status PENDING, got %s", payment.Status)
	}
	if payment.WalletID != "wallet-1" {
		t.Errorf("expected wallet_id wallet-1, got %s", payment.WalletID)
	}
	if payment.Currency != "ARS" {
		t.Errorf("expected currency ARS, got %s", payment.Currency)
	}
}

func TestCreatePayment_WalletNotFound(t *testing.T) {
	walletRepo, _, _, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-unknown").
		Return(nil, domain.ErrWalletNotFound)

	payment, err := uc.Execute(context.Background(), usecase.CreatePaymentInput{
		UserID: "user-unknown",
		Amount: 50.00,
	})

	if payment != nil {
		t.Errorf("expected nil payment, got %+v", payment)
	}
	if !errors.Is(err, domain.ErrWalletNotFound) {
		t.Errorf("expected ErrWalletNotFound, got %v", err)
	}
}

func TestCreatePayment_InsufficientFunds(t *testing.T) {
	walletRepo, _, _, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(&domain.Wallet{
			ID:      "wallet-1",
			UserID:  "user-1",
			Balance: 10.00,
		}, nil)

	payment, err := uc.Execute(context.Background(), usecase.CreatePaymentInput{
		UserID: "user-1",
		Amount: 999.00,
	})

	if payment != nil {
		t.Errorf("expected nil payment, got %+v", payment)
	}
	if !errors.Is(err, domain.ErrInsufficientFunds) {
		t.Errorf("expected ErrInsufficientFunds, got %v", err)
	}
}

func TestCreatePayment_CreateFails(t *testing.T) {
	walletRepo, paymentRepo, _, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(&domain.Wallet{
			ID:      "wallet-1",
			UserID:  "user-1",
			Balance: 500.00,
		}, nil)

	dbErr := errors.New("connection refused")
	paymentRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(dbErr)

	payment, err := uc.Execute(context.Background(), usecase.CreatePaymentInput{
		UserID: "user-1",
		Amount: 100.00,
	})

	if payment != nil {
		t.Errorf("expected nil payment, got %+v", payment)
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}

func TestCreatePayment_PublishFails(t *testing.T) {
	walletRepo, paymentRepo, publisher, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(&domain.Wallet{
			ID:      "wallet-1",
			UserID:  "user-1",
			Balance: 500.00,
		}, nil)

	paymentRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p *domain.Payment) error {
			p.ID = "pay-1"
			return nil
		})

	publishErr := errors.New("kafka unavailable")
	publisher.EXPECT().
		Publish(gomock.Any(), gomock.Any()).
		Return(publishErr)

	payment, err := uc.Execute(context.Background(), usecase.CreatePaymentInput{
		UserID: "user-1",
		Amount: 100.00,
	})

	if payment != nil {
		t.Errorf("expected nil payment, got %+v", payment)
	}
	if !errors.Is(err, publishErr) {
		t.Errorf("expected kafka error, got %v", err)
	}
}

func TestCreatePayment_WalletRepositoryError(t *testing.T) {
	walletRepo, _, _, uc := setupUseCase(t)

	repoErr := errors.New("unexpected db error")
	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(nil, repoErr)

	payment, err := uc.Execute(context.Background(), usecase.CreatePaymentInput{
		UserID: "user-1",
		Amount: 50.00,
	})

	if payment != nil {
		t.Errorf("expected nil payment, got %+v", payment)
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}
