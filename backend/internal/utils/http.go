package utils

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type APIResponse struct {
	Success   bool   `json:"success"`
	Data      any    `json:"data,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// writeJSON encodes the body. A failure here means the client went away mid-response;
// there is nothing to send back, so it is only logged at debug level.
func writeJSON(w http.ResponseWriter, body APIResponse) {
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Debug("failed to write response", "error", err)
	}
}

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	writeJSON(w, APIResponse{
		Success: true,
		Data:    data,
	})
}

func RespondSuccess(w http.ResponseWriter, status int, msg string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	writeJSON(w, APIResponse{
		Success: true,
		Message: msg,
		Data:    data,
	})
}

func RespondError(w http.ResponseWriter, status int, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	writeJSON(w, APIResponse{
		Success: false,
		Error:   errMsg,
	})
}
