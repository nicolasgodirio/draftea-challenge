package handler_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"draftea-challenge/getbalances/internal/domain"
	"draftea-challenge/getbalances/internal/infra/handler"
	"draftea-challenge/getbalances/internal/infra/handler/dto"
	"draftea-challenge/getbalances/internal/infra/handler/mocks"
)

func fakeJWT(userID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, _ := json.Marshal(map[string]string{"sub": userID})
	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	return fmt.Sprintf("%s.%s.%s", header, body, sig)
}

func setupRouter(t *testing.T) (*mocks.MockBalanceGetter, http.Handler) {
	ctrl := gomock.NewController(t)
	mockUC := mocks.NewMockBalanceGetter(ctrl)
	h := handler.New(mockUC)
	router := handler.NewRouter(h)
	return mockUC, router
}

func TestHandle_Success(t *testing.T) {
	mockUC, router := setupRouter(t)

	mockUC.EXPECT().
		Execute(gomock.Any(), "user-123").
		Return(&domain.Wallet{
			ID:       "wallet-1",
			UserID:   "user-123",
			Balance:  500.00,
			Currency: "USD",
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/balances", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	var resp dto.BalanceResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.UserID != "user-123" {
		t.Errorf("expected user_id user-123, got %s", resp.UserID)
	}
	if resp.Balance != 500.00 {
		t.Errorf("expected balance 500.00, got %f", resp.Balance)
	}
	if resp.Currency != "USD" {
		t.Errorf("expected currency USD, got %s", resp.Currency)
	}
}

func TestHandle_MissingAuthToken(t *testing.T) {
	_, router := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/balances", nil)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestHandle_InvalidToken(t *testing.T) {
	_, router := setupRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/balances", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/balances", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/balances", nil)
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}
