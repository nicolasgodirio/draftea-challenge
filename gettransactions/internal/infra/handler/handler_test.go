package handler_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"draftea-challenge/gettransactions/internal/domain"
	"draftea-challenge/gettransactions/internal/infra/handler"
	"draftea-challenge/gettransactions/internal/infra/handler/dto"
	"draftea-challenge/gettransactions/internal/infra/handler/mocks"
)

func fakeJWT(userID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, _ := json.Marshal(map[string]string{"sub": userID})
	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	return fmt.Sprintf("%s.%s.%s", header, body, sig)
}

func setupRouter(t *testing.T) (*mocks.MockTransactionsGetter, http.Handler) {
	ctrl := gomock.NewController(t)
	mockUC := mocks.NewMockTransactionsGetter(ctrl)
	h := handler.New(mockUC)
	router := handler.NewRouter(h)
	return mockUC, router
}

func TestHandle_Success(t *testing.T) {
	mockUC, router := setupRouter(t)

	refID := "pay-1"
	now := time.Now().Truncate(time.Second)

	mockUC.EXPECT().
		Execute(gomock.Any(), "user-123").
		Return([]domain.Transaction{
			{
				ID:          "txn-1",
				WalletID:    "wallet-1",
				Type:        domain.TransactionTypeDebit,
				Amount:      100.50,
				ReferenceID: &refID,
				Description: "Payment debit",
				CreatedAt:   now,
			},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp dto.TransactionsResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.UserID != "user-123" {
		t.Errorf("expected user_id user-123, got %s", resp.UserID)
	}
	if len(resp.Transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(resp.Transactions))
	}
	if resp.Transactions[0].ID != "txn-1" {
		t.Errorf("expected transaction id txn-1, got %s", resp.Transactions[0].ID)
	}
	if resp.Transactions[0].Type != "DEBIT" {
		t.Errorf("expected type DEBIT, got %s", resp.Transactions[0].Type)
	}
	if resp.Transactions[0].Amount != 100.50 {
		t.Errorf("expected amount 100.50, got %f", resp.Transactions[0].Amount)
	}
}

func TestHandle_EmptyTransactions(t *testing.T) {
	mockUC, router := setupRouter(t)

	mockUC.EXPECT().
		Execute(gomock.Any(), "user-123").
		Return([]domain.Transaction{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp dto.TransactionsResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Transactions) != 0 {
		t.Errorf("expected 0 transactions, got %d", len(resp.Transactions))
	}
}

func TestHandle_MissingAuthToken(t *testing.T) {
	_, router := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestHandle_InvalidToken(t *testing.T) {
	_, router := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestHandle_WalletNotFound(t *testing.T) {
	mockUC, router := setupRouter(t)

	mockUC.EXPECT().
		Execute(gomock.Any(), "user-999").
		Return(nil, domain.ErrWalletNotFound)

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-999"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandle_InternalServerError(t *testing.T) {
	mockUC, router := setupRouter(t)

	mockUC.EXPECT().
		Execute(gomock.Any(), "user-123").
		Return(nil, fmt.Errorf("unexpected db error"))

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}
