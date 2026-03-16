package usecase_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"draftea-challenge/getbalances/internal/domain"
	"draftea-challenge/getbalances/internal/usecase"
	"draftea-challenge/getbalances/internal/usecase/mocks"
)

func setupUseCase(t *testing.T) (*mocks.MockWalletRepository, *usecase.GetBalance) {
	ctrl := gomock.NewController(t)
	walletRepo := mocks.NewMockWalletRepository(ctrl)
	uc := usecase.NewGetBalance(walletRepo)
	return walletRepo, uc
}

func TestGetBalance_Success(t *testing.T) {
	walletRepo, uc := setupUseCase(t)

	expected := &domain.Wallet{
		ID:       "wallet-1",
		UserID:   "user-1",
		Balance:  10000.00,
		Currency: "ARS",
	}

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(expected, nil)

	wallet, err := uc.Execute(context.Background(), "user-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if wallet == nil {
		t.Fatal("expected wallet, got nil")
	}
	if wallet.ID != "wallet-1" {
		t.Errorf("expected wallet_id wallet-1, got %s", wallet.ID)
	}
	if wallet.Balance != 10000.00 {
		t.Errorf("expected balance 10000.00, got %f", wallet.Balance)
	}
	if wallet.Currency != "ARS" {
		t.Errorf("expected currency ARS, got %s", wallet.Currency)
	}
}

func TestGetBalance_WalletNotFound(t *testing.T) {
	walletRepo, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-unknown").
		Return(nil, domain.ErrWalletNotFound)

	wallet, err := uc.Execute(context.Background(), "user-unknown")

	if wallet != nil {
		t.Errorf("expected nil wallet, got %+v", wallet)
	}
	if !errors.Is(err, domain.ErrWalletNotFound) {
		t.Errorf("expected ErrWalletNotFound, got %v", err)
	}
}

func TestGetBalance_RepositoryError(t *testing.T) {
	walletRepo, uc := setupUseCase(t)

	repoErr := errors.New("connection timeout")
	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(nil, repoErr)

	wallet, err := uc.Execute(context.Background(), "user-1")

	if wallet != nil {
		t.Errorf("expected nil wallet, got %+v", wallet)
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}
