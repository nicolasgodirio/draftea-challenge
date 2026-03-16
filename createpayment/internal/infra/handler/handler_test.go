package handler_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"draftea-challenge/createpayment/internal/domain"
	"draftea-challenge/createpayment/internal/infra/handler"
	"draftea-challenge/createpayment/internal/infra/handler/dto"
	"draftea-challenge/createpayment/internal/infra/handler/mocks"
	"draftea-challenge/createpayment/internal/usecase"
)

func fakeJWT(userID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, _ := json.Marshal(map[string]string{"sub": userID})
	body := base64.RawURLEncoding.EncodeToString(payload)
	sig := base64.RawURLEncoding.EncodeToString([]byte("sig"))
	return fmt.Sprintf("%s.%s.%s", header, body, sig)
}

func setupRouter(t *testing.T) (*mocks.MockPaymentCreator, http.Handler) {
	ctrl := gomock.NewController(t)
	mockUC := mocks.NewMockPaymentCreator(ctrl)
	h := handler.New(mockUC)
	router := handler.NewRouter(h)
	return mockUC, router
}

func TestHandle_Success(t *testing.T) {
	mockUC, router := setupRouter(t)

	mockUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ interface{}, input usecase.CreatePaymentInput) (*domain.Payment, error) {
			if input.UserID != "user-123" {
				t.Errorf("expected user_id user-123, got %s", input.UserID)
			}
			if input.Amount != 100.50 {
				t.Errorf("expected amount 100.50, got %f", input.Amount)
			}
			return &domain.Payment{
				ID:     "pay-1",
				Status: domain.PaymentStatusPending,
			}, nil
		})

	body := `{"amount": 100.50}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", rr.Code)
	}

	var resp dto.CreatePaymentResponse
	json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Status != "PENDING" {
		t.Errorf("expected status PENDING, got %s", resp.Status)
	}
}

func TestHandle_MissingAuthToken(t *testing.T) {
	_, router := setupRouter(t)

	body := `{"amount": 100}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestHandle_InvalidToken(t *testing.T) {
	_, router := setupRouter(t)

	body := `{"amount": 100}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer not-a-jwt")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestHandle_InvalidRequestBody(t *testing.T) {
	_, router := setupRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString("not json"))
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandle_InvalidAmount(t *testing.T) {
	_, router := setupRouter(t)

	body := `{"amount": -10}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandle_ZeroAmount(t *testing.T) {
	_, router := setupRouter(t)

	body := `{"amount": 0}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestHandle_WalletNotFound(t *testing.T) {
	mockUC, router := setupRouter(t)

	mockUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(nil, domain.ErrWalletNotFound)

	body := `{"amount": 100}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-999"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rr.Code)
	}
}

func TestHandle_InsufficientFunds(t *testing.T) {
	mockUC, router := setupRouter(t)

	mockUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(nil, domain.ErrInsufficientFunds)

	body := `{"amount": 999999}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", rr.Code)
	}
}

func TestHandle_InternalServerError(t *testing.T) {
	mockUC, router := setupRouter(t)

	mockUC.EXPECT().
		Execute(gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("unexpected db error"))

	body := `{"amount": 100}`
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+fakeJWT("user-123"))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rr.Code)
	}
}
