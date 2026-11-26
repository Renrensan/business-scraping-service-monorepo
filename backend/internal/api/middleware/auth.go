package middleware

import (
	"context"
	"net/http"
	"strings"

	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

// SessionContextKey holds the request's session in its context.
const SessionContextKey contextKey = "session_data"

// SessionCookieName is the cookie that carries the session token.
const SessionCookieName = "bizscrape_session"

func AuthMiddleware(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				utils.RespondError(w, http.StatusUnauthorized, "Unauthorized: session required")
				return
			}

			sessionData, err := authSvc.ValidateSession(r.Context(), token)
			if err != nil || sessionData == nil {
				utils.RespondError(w, http.StatusUnauthorized, "Unauthorized: session invalid or expired")
				return
			}

			ctx := context.WithValue(r.Context(), SessionContextKey, sessionData)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetSessionFromContext(ctx context.Context) *model.SessionData {
	if ctx == nil {
		return nil
	}
	if s, ok := ctx.Value(SessionContextKey).(*model.SessionData); ok {
		return s
	}
	return nil
}

func extractToken(r *http.Request) string {
	if cookie, err := r.Cookie(SessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ""
}
