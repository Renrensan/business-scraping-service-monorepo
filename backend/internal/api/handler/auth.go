package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"business-scraping-service/backend/internal/api/middleware"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type AuthHandler struct {
	authSvc *service.AuthService
	audit   *middleware.Recorder
}

func NewAuthHandler(authSvc *service.AuthService, audit *middleware.Recorder) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, audit: audit}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	admin, token, err := h.authSvc.Login(r.Context(), req.UsernameOrEmail, req.Password)
	if err != nil {
		// Logins happen before there is a session, so they are audited here, not by the middleware.
		attempted := strings.TrimSpace(req.UsernameOrEmail)
		if len(attempted) > 100 {
			attempted = attempted[:100]
		}
		h.audit.Record(r, &model.AuditEntry{
			AdminName: attempted, Action: "auth.login_failed", Summary: "Failed login",
			Method: r.Method, Route: "/auth/login", Status: http.StatusUnauthorized,
		})
		utils.RespondError(w, http.StatusUnauthorized, "Invalid username/email or password")
		return
	}
	id := admin.ID
	h.audit.Record(r, &model.AuditEntry{
		AdminID: &id, AdminName: admin.Username, Action: "auth.login", Summary: "Logged in",
		EntityType: "admin", EntityID: admin.ID.String(), Method: r.Method, Route: "/auth/login", Status: http.StatusOK,
	})

	// Set HttpOnly session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // set to true in production over HTTPS
	})

	utils.RespondJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":        admin.ID,
			"username":  admin.Username,
			"email":     admin.Email,
			"createdAt": admin.CreatedAt,
		},
		"token": token,
	})
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSessionFromContext(r.Context())
	if session == nil {
		utils.RespondError(w, http.StatusUnauthorized, "No active session")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]any{
		"adminId":      session.AdminID,
		"username":     session.Username,
		"email":        session.Email,
		"createdAt":    session.CreatedAt,
		"lastActiveAt": session.LastActiveAt,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSessionFromContext(r.Context())
	if session != nil {
		if err := h.authSvc.Logout(r.Context(), session.Token); err != nil {
			// The cookie is cleared below either way; the session expires on its own TTL.
			slog.Warn("failed to delete session on logout", "error", err)
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		MaxAge:   -1,
	})

	utils.RespondSuccess(w, http.StatusOK, "Logged out successfully", nil)
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	session := middleware.GetSessionFromContext(r.Context())
	if session == nil {
		utils.RespondError(w, http.StatusUnauthorized, "Session required")
		return
	}

	var req model.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(req.NewPassword) < service.MinPasswordLength {
		utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("New password must be at least %d characters", service.MinPasswordLength))
		return
	}

	if err := h.authSvc.ChangePassword(r.Context(), session.AdminID, req.CurrentPassword, req.NewPassword); err != nil {
		respondErr(w, r, err, "Failed to change password")
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Password changed successfully", nil)
}
