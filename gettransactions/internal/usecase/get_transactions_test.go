package usecase_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"draftea-challenge/gettransactions/internal/domain"
	"draftea-challenge/gettransactions/internal/usecase"
	"draftea-challenge/gettransactions/internal/usecase/mocks"
)

func setupUseCase(t *testing.T) (*mocks.MockWalletRepository, *usecase.GetTransactions) {
	ctrl := gomock.NewController(t)
	walletRepo := mocks.NewMockWalletRepository(ctrl)
	uc := usecase.NewGetTransactions(walletRepo)
	return walletRepo, uc
}

func TestGetTransactions_Success(t *testing.T) {
	walletRepo, uc := setupUseCase(t)

	refID := "pay-1"

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(&domain.Wallet{ID: "wallet-1", UserID: "user-1"}, nil)

	walletRepo.EXPECT().
		GetTransactionsByUserID(gomock.Any(), "user-1").
		Return([]domain.Transaction{
			{
				ID:          "txn-1",
				WalletID:    "wallet-1",
				Type:        domain.TransactionTypeDebit,
				Amount:      100.00,
				ReferenceID: &refID,
				Description: "Payment debit",
			},
			{
				ID:          "txn-2",
				WalletID:    "wallet-1",
				Type:        domain.TransactionTypeCredit,
				Amount:      10000.00,
				Description: "Initial deposit",
			},
		}, nil)

	txns, err := uc.Execute(context.Background(), "user-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(txns) != 2 {
		t.Fatalf("expected 2 transactions, got %d", len(txns))
	}
	if txns[0].ID != "txn-1" {
		t.Errorf("expected first transaction txn-1, got %s", txns[0].ID)
	}
	if txns[0].Type != domain.TransactionTypeDebit {
		t.Errorf("expected type DEBIT, got %s", txns[0].Type)
	}
	if txns[0].ReferenceID == nil || *txns[0].ReferenceID != "pay-1" {
		t.Errorf("expected reference_id pay-1, got %v", txns[0].ReferenceID)
	}
}

func TestGetTransactions_EmptyResult(t *testing.T) {
	walletRepo, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(&domain.Wallet{ID: "wallet-1", UserID: "user-1"}, nil)

	walletRepo.EXPECT().
		GetTransactionsByUserID(gomock.Any(), "user-1").
		Return([]domain.Transaction{}, nil)

	txns, err := uc.Execute(context.Background(), "user-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(txns) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(txns))
	}
}

func TestGetTransactions_WalletNotFound(t *testing.T) {
	walletRepo, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-unknown").
		Return(nil, domain.ErrWalletNotFound)

	txns, err := uc.Execute(context.Background(), "user-unknown")

	if txns != nil {
		t.Errorf("expected nil transactions, got %+v", txns)
	}
	if !errors.Is(err, domain.ErrWalletNotFound) {
		t.Errorf("expected ErrWalletNotFound, got %v", err)
	}
}

func TestGetTransactions_WalletRepositoryError(t *testing.T) {
	walletRepo, uc := setupUseCase(t)

	repoErr := errors.New("connection timeout")
	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(nil, repoErr)

	txns, err := uc.Execute(context.Background(), "user-1")

	if txns != nil {
		t.Errorf("expected nil transactions, got %+v", txns)
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

func TestGetTransactions_GetTransactionsFails(t *testing.T) {
	walletRepo, uc := setupUseCase(t)

	walletRepo.EXPECT().
		GetByUserID(gomock.Any(), "user-1").
		Return(&domain.Wallet{ID: "wallet-1", UserID: "user-1"}, nil)

	repoErr := errors.New("query failed")
	walletRepo.EXPECT().
		GetTransactionsByUserID(gomock.Any(), "user-1").
		Return(nil, repoErr)

	txns, err := uc.Execute(context.Background(), "user-1")

	if txns != nil {
		t.Errorf("expected nil transactions, got %+v", txns)
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("expected query error, got %v", err)
	}
}
