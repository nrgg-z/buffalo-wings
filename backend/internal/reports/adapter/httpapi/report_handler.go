package httpapi

import (
	"context"
	"errors"
	"log"
	"net/http"

	authhttp "buffalo-pos/internal/auth/adapter/httpapi"
	authdomain "buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/platform/httpx"
	"buffalo-pos/internal/reports/domain"
)

type ReportService interface {
	GetDailySummary(ctx context.Context, rawDate string) (*domain.DailySummary, error)
	ListVoids(ctx context.Context, rawDate string) (string, []domain.VoidDetail, error)
}

type ReportHandler struct {
	reports ReportService
}

func NewReportHandler(reports ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

// Register registra las rutas de reportes. Solo MANAGER y ADMIN.
func (h *ReportHandler) Register(mux *http.ServeMux, v authhttp.TokenVerifier) {
	manager := authhttp.Protected(v, authdomain.RoleManager)

	mux.Handle("GET /api/v1/reports/daily-summary", manager(http.HandlerFunc(h.dailySummary)))
	mux.Handle("GET /api/v1/reports/voids", manager(http.HandlerFunc(h.listVoids)))
}

func (h *ReportHandler) dailySummary(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	summary, err := h.reports.GetDailySummary(r.Context(), date)
	if errors.Is(err, domain.ErrInvalidDate) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_date", err.Error())
		return
	}
	if err != nil {
		log.Printf("error generando reporte diario: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "No se pudo generar el reporte diario")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}

func (h *ReportHandler) listVoids(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	resolvedDate, voids, err := h.reports.ListVoids(r.Context(), date)
	if errors.Is(err, domain.ErrInvalidDate) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_date", err.Error())
		return
	}
	if err != nil {
		log.Printf("error consultando anulaciones: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "No se pudieron consultar las anulaciones")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"date": resolvedDate, "voids": voids})
}
