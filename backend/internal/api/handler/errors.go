package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"business-scraping-service/backend/internal/api/middleware"
	"business-scraping-service/backend/internal/apperr"
	"business-scraping-service/backend/internal/utils"
)

// respondErr sends an error to the client. Errors the user can act on (see package apperr) are
// sent with their message and matching status; anything else is a server fault and goes
// through serverError.
func respondErr(w http.ResponseWriter, r *http.Request, err error, fallback string) {
	switch {
	case errors.Is(err, apperr.ErrInvalid):
		utils.RespondError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, apperr.ErrNotFound):
		utils.RespondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, apperr.ErrConflict):
		utils.RespondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, apperr.ErrUnavailable):
		utils.RespondError(w, http.StatusTooManyRequests, err.Error())
	default:
		serverError(w, r, fallback, err)
	}
}

// serverError logs the real error with the request id and sends the client only msg and that id,
// so database and internal details never reach the browser but a report can still be traced.
func serverError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	reqID, _ := r.Context().Value(middleware.RequestIDKey).(string)
	slog.Error(msg, "error", err, "request_id", reqID, "method", r.Method, "path", r.URL.Path)
	if reqID != "" {
		msg = fmt.Sprintf("%s (reference %s)", msg, reqID)
	}
	utils.RespondError(w, http.StatusInternalServerError, msg)
}
