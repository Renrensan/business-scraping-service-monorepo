package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"business-scraping-service/backend/internal/service"
	"business-scraping-service/backend/internal/utils"
)

type ExportHandler struct {
	exportSvc *service.ExportService
}

func NewExportHandler(exportSvc *service.ExportService) *ExportHandler {
	return &ExportHandler{exportSvc: exportSvc}
}

func (h *ExportHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	filter := parseBusinessFilter(r.URL.Query())

	csvBytes, _, err := h.exportSvc.ExportCSV(r.Context(), filter)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate CSV: "+err.Error())
		return
	}

	filename := fmt.Sprintf("business_leads_%s.csv", time.Now().Format("2006-01-02_150405"))

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(csvBytes)
}

// ExportWAContacts downloads the filtered leads as a Google Contacts CSV for WhatsApp.
// Query params (besides the normal filters):
//   mobile_only   default true  — skip leads that only have a landline
//   status_prefix default false — name contacts "[HOT] Business Name"
func (h *ExportHandler) ExportWAContacts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := parseBusinessFilter(q)
	opts := service.WAExportOptions{
		MobileOnly:   q.Get("mobile_only") != "false",
		StatusPrefix: q.Get("status_prefix") == "true",
	}

	res, err := h.exportSvc.ExportWAContacts(r.Context(), filter, opts)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "Failed to generate WhatsApp contacts: "+err.Error())
		return
	}

	filename := fmt.Sprintf("wa_contacts_%s.csv", time.Now().Format("2006-01-02_150405"))

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	w.Header().Set("X-Export-Count", strconv.Itoa(res.Exported))
	w.Header().Set("X-Export-Skipped", strconv.Itoa(res.Skipped))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(res.CSV)
}
