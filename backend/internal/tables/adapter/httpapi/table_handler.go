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
	"buffalo-pos/internal/platform/httpx"
	"buffalo-pos/internal/tables/domain"
	"buffalo-pos/internal/tables/port"
)

// maxBodyBytes limita el tamaño del cuerpo de las peticiones.
const maxBodyBytes = 1 << 10

// TableService es lo que el handler necesita del servicio de mesas.
type TableService interface {
	List(ctx context.Context) ([]domain.Table, error)
	Assign(ctx context.Context, ref port.TableRef, guests int) (*domain.Table, error)
	Merge(ctx context.Context, refs []port.TableRef) (*domain.Table, error)
	MarkClean(ctx context.Context, ref port.TableRef) (*domain.Table, error)
}

// TableHandler expone las rutas HTTP de mesas.
type TableHandler struct {
	tables TableService
}

func NewTableHandler(tables TableService) *TableHandler {
	return &TableHandler{tables: tables}
}

// Register agrega las rutas de mesas al router. ADMIN siempre pasa.
func (h *TableHandler) Register(mux *http.ServeMux, v authhttp.TokenVerifier) {
	view := authhttp.Protected(v, authdomain.RoleHostess, authdomain.RoleWaiter, authdomain.RoleCashier, authdomain.RoleManager)
	seat := authhttp.Protected(v, authdomain.RoleHostess, authdomain.RoleManager)
	clean := authhttp.Protected(v, authdomain.RoleWaiter, authdomain.RoleHostess, authdomain.RoleManager)

	mux.Handle("GET /api/v1/tables", view(http.HandlerFunc(h.list)))
	mux.Handle("POST /api/v1/tables/merge", seat(http.HandlerFunc(h.merge)))
	mux.Handle("POST /api/v1/tables/{id}/assign", seat(http.HandlerFunc(h.assign)))
	mux.Handle("POST /api/v1/tables/{id}/clean", clean(http.HandlerFunc(h.clean)))
}

type tableResponse struct {
	ID                int64  `json:"id"`
	Code              string `json:"code"`
	Zone              string `json:"zone"`
	Capacity          int    `json:"capacity"`
	EffectiveCapacity int    `json:"effective_capacity"`
	Status            string `json:"status"`
	CurrentGuests     int    `json:"current_guests"`
	AssignedWaiterID  *int64 `json:"assigned_waiter_id"`
	MergedIntoID      *int64 `json:"merged_into_id"`
	Version           uint32 `json:"version"`
}

func toResponse(t domain.Table) tableResponse {
	return tableResponse{
		ID:                t.ID,
		Code:              t.Code,
		Zone:              t.Zone,
		Capacity:          t.Capacity,
		EffectiveCapacity: t.EffectiveCapacity,
		Status:            t.Status.APIName(),
		CurrentGuests:     t.CurrentGuests,
		AssignedWaiterID:  t.AssignedWaiterID,
		MergedIntoID:      t.MergedIntoID,
		Version:           t.Version,
	}
}

type versionRequest struct {
	Version uint32 `json:"version"`
}

type assignRequest struct {
	Guests  int    `json:"guests"`
	Version uint32 `json:"version"`
}

type mergeRequest struct {
	Tables []struct {
		ID      int64  `json:"id"`
		Version uint32 `json:"version"`
	} `json:"tables"`
}

func (h *TableHandler) list(w http.ResponseWriter, r *http.Request) {
	tables, err := h.tables.List(r.Context())
	if err != nil {
		writeError(w, "listar mesas", err)
		return
	}
	out := make([]tableResponse, 0, len(tables))
	for _, t := range tables {
		out = append(out, toResponse(t))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"tables": out})
}

func (h *TableHandler) assign(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req assignRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Version == 0 {
		badRequest(w, "Se requiere version")
		return
	}
	t, err := h.tables.Assign(r.Context(), port.TableRef{ID: id, Version: req.Version}, req.Guests)
	if err != nil {
		writeError(w, "asignar mesa", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(*t))
}

func (h *TableHandler) merge(w http.ResponseWriter, r *http.Request) {
	var req mergeRequest
	if !decode(w, r, &req) {
		return
	}
	refs := make([]port.TableRef, 0, len(req.Tables))
	for _, t := range req.Tables {
		if t.ID <= 0 || t.Version == 0 {
			badRequest(w, "Cada mesa requiere id y version")
			return
		}
		refs = append(refs, port.TableRef{ID: t.ID, Version: t.Version})
	}
	t, err := h.tables.Merge(r.Context(), refs)
	if err != nil {
		writeError(w, "combinar mesas", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(*t))
}

func (h *TableHandler) clean(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var req versionRequest
	if !decode(w, r, &req) {
		return
	}
	if req.Version == 0 {
		badRequest(w, "Se requiere version")
		return
	}
	t, err := h.tables.MarkClean(r.Context(), port.TableRef{ID: id, Version: req.Version})
	if err != nil {
		writeError(w, "marcar mesa limpia", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toResponse(*t))
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		badRequest(w, "El id de la mesa no es válido")
		return 0, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		badRequest(w, "El cuerpo de la petición no es válido")
		return false
	}
	return true
}

func badRequest(w http.ResponseWriter, msg string) {
	httpx.WriteError(w, http.StatusBadRequest, "invalid_request", msg)
}

// writeError traduce los errores del módulo al formato estándar de la API.
func writeError(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, port.ErrTableNotFound):
		httpx.WriteError(w, http.StatusNotFound, "table_not_found", "La mesa no existe")
	case errors.Is(err, port.ErrVersionConflict):
		httpx.WriteError(w, http.StatusConflict, "version_conflict", "La mesa cambió; actualiza el mapa e intenta de nuevo")
	case errors.Is(err, domain.ErrInvalidGuests):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_guests", "La cantidad de comensales debe ser al menos 1")
	case errors.Is(err, domain.ErrOverCapacity):
		httpx.WriteError(w, http.StatusBadRequest, "over_capacity", "Los comensales superan la capacidad de la mesa")
	case errors.Is(err, domain.ErrInvalidMerge):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_merge", "Indica al menos 2 mesas distintas")
	case errors.Is(err, domain.ErrTableNotFree):
		httpx.WriteError(w, http.StatusConflict, "table_not_free", "La mesa no está libre")
	case errors.Is(err, domain.ErrTableMerged):
		httpx.WriteError(w, http.StatusConflict, "table_merged", "La mesa está combinada con otra")
	case errors.Is(err, domain.ErrNotCleaning):
		httpx.WriteError(w, http.StatusConflict, "table_not_cleaning", "La mesa no está por limpiar")
	case errors.Is(err, domain.ErrMergeNotAllowed):
		httpx.WriteError(w, http.StatusConflict, "merge_not_allowed",
			"Solo se combinan mesas libres de la misma zona con códigos consecutivos")
	default:
		log.Printf("error al %s: %v", op, err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Error interno del servidor")
	}
}
