package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	authhttp "buffalo-pos/internal/auth/adapter/httpapi"
	authdomain "buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/kds/domain"
	"buffalo-pos/internal/platform/httpx"
)

const maxBodyBytes = 1 << 10

type KDSService interface {
	ListQueue(ctx context.Context, station string) ([]domain.KDSItem, error)
	ListReady(ctx context.Context) ([]domain.KDSItem, error)
	UpdateItemStatus(ctx context.Context, itemID int64, nextStatus string) (*domain.KDSItem, error)
}

type KDSHandler struct {
	kds KDSService
}

func NewKDSHandler(kds KDSService) *KDSHandler {
	return &KDSHandler{kds: kds}
}

// Register agrega las rutas de cocina (KDS) y avisos de platillos listos. ADMIN siempre pasa.
func (h *KDSHandler) Register(mux *http.ServeMux, v authhttp.TokenVerifier) {
	kitchen := authhttp.Protected(v, authdomain.RoleKitchen, authdomain.RoleManager)
	shared := authhttp.Protected(v, authdomain.RoleKitchen, authdomain.RoleWaiter, authdomain.RoleManager)

	mux.Handle("GET /api/v1/kds/queue", kitchen(http.HandlerFunc(h.listQueue)))
	mux.Handle("GET /api/v1/kds/ready", shared(http.HandlerFunc(h.listReady)))
	mux.Handle("PATCH /api/v1/kds/items/{id}/status", shared(http.HandlerFunc(h.updateStatus)))
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func (h *KDSHandler) listQueue(w http.ResponseWriter, r *http.Request) {
	station := r.URL.Query().Get("station")
	items, err := h.kds.ListQueue(r.Context(), station)
	if errors.Is(err, domain.ErrInvalidStation) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_station", err.Error())
		return
	}
	if err != nil {
		log.Printf("error consultando cola KDS: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "No se pudo consultar la cola de cocina")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *KDSHandler) listReady(w http.ResponseWriter, r *http.Request) {
	items, err := h.kds.ListReady(r.Context())
	if err != nil {
		log.Printf("error consultando platillos listos: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "No se pudieron consultar los platillos listos")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *KDSHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de ítem inválido")
		return
	}

	var req updateStatusRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "Se requiere el campo status")
		return
	}

	item, err := h.kds.UpdateItemStatus(r.Context(), id, req.Status)
	switch {
	case errors.Is(err, domain.ErrInvalidStatus):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_status", err.Error())
	case errors.Is(err, domain.ErrItemNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domain.ErrInvalidTransition):
		httpx.WriteError(w, http.StatusConflict, "invalid_transition", err.Error())
	case err != nil:
		log.Printf("error actualizando estado KDS del ítem %d: %v", id, err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "No se pudo actualizar el estado")
	default:
		httpx.WriteJSON(w, http.StatusOK, item)
	}
}
