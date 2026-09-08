package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"business-scraping-service/backend/internal/api/middleware"
	"business-scraping-service/backend/internal/model"
	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type AuthHandler struct {
	authSvc *service.AuthService
}

func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	admin, token, err := h.authSvc.Login(r.Context(), req.UsernameOrEmail, req.Password)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "Invalid username/email or password")
		return
	}

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

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
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

	utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
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
		_ = h.authSvc.Logout(r.Context(), session.Token)
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

	if req.NewPassword == "" || len(req.NewPassword) < 6 {
		utils.RespondError(w, http.StatusBadRequest, "New password must be at least 6 characters")
		return
	}

	if err := h.authSvc.ChangePassword(r.Context(), session.AdminID, req.CurrentPassword, req.NewPassword); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	utils.RespondSuccess(w, http.StatusOK, "Password changed successfully", nil)
}
