package repository

import (
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
)

// decodeJSON decodes a JSONB column into dst. NULL (no bytes) leaves dst unchanged.
func decodeJSON(raw []byte, dst any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, dst)
}

// decodeForDisplay decodes a JSONB column that is only being read for display. A value of the
// wrong shape is logged and the field left empty, so one bad row doesn't fail a whole page.
// Code that writes the value back must use decodeJSON and handle the error instead.
func decodeForDisplay(raw []byte, dst any, table string, id uuid.UUID, column string) {
	if err := decodeJSON(raw, dst); err != nil {
		slog.Warn("unreadable JSON column, leaving it empty",
			"table", table, "id", id, "column", column, "error", err)
	}
}
