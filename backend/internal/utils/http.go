package utils

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Success   bool   `json:"success"`
	Data      any    `json:"data,omitempty"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func RespondSuccess(w http.ResponseWriter, status int, msg string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Message: msg,
		Data:    data,
	})
}

func RespondError(w http.ResponseWriter, status int, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   errMsg,
	})
}
