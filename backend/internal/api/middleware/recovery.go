package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"business-scraping-service/backend/internal/utils"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("Unhandled Panic Recovered",
					"error", rec,
					"stack", string(debug.Stack()),
					"path", r.URL.Path,
				)
				utils.RespondError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
