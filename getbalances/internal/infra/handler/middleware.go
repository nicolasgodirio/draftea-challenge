package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"draftea-challenge/getbalances/internal/domain"
)

type contextKey string

const userIDKey contextKey = "user_id"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			slog.WarnContext(r.Context(), "missing or invalid authorization header")
			handleError(w, domain.ErrUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		userID, err := extractUserIDFromToken(token)
		if err != nil {
			slog.WarnContext(r.Context(), "failed to extract user_id from token", "error", err)
			handleError(w, domain.ErrUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		slog.InfoContext(ctx, "request authenticated", "user_id", userID, "method", r.Method, "path", r.URL.Path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok && userID != ""
}

func extractUserIDFromToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errInvalidToken
	}

	var claims struct {
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", errInvalidToken
	}

	if claims.Sub == "" {
		return "", errInvalidToken
	}

	return claims.Sub, nil
}

var errInvalidToken = errors.New("invalid token")
