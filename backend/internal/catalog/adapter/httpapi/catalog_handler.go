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
	"buffalo-pos/internal/catalog/domain"
	"buffalo-pos/internal/platform/httpx"
)

const maxBodyBytes = 1 << 10

type CatalogService interface {
	GetCatalog(ctx context.Context) ([]domain.Category, error)
	SetProductAvailability(ctx context.Context, productID int64, available bool) (*domain.Product, error)
	SetModifierAvailability(ctx context.Context, optionID int64, available bool) (*domain.ModifierOption, error)
}

type CatalogHandler struct {
	catalog CatalogService
}

func NewCatalogHandler(catalog CatalogService) *CatalogHandler {
	return &CatalogHandler{catalog: catalog}
}

// Register agrega las rutas del catálogo. ADMIN siempre pasa.
func (h *CatalogHandler) Register(mux *http.ServeMux, v authhttp.TokenVerifier) {
	view := authhttp.Protected(v,
		authdomain.RoleWaiter,
		authdomain.RoleCashier,
		authdomain.RoleKitchen,
		authdomain.RoleHostess,
		authdomain.RoleManager,
	)
	manage := authhttp.Protected(v, authdomain.RoleKitchen, authdomain.RoleManager)

	mux.Handle("GET /api/v1/catalog", view(http.HandlerFunc(h.getCatalog)))
	mux.Handle("PATCH /api/v1/catalog/products/{id}/availability", manage(http.HandlerFunc(h.setProductAvailability)))
	mux.Handle("PATCH /api/v1/catalog/modifiers/{id}/availability", manage(http.HandlerFunc(h.setModifierAvailability)))
}

type availabilityRequest struct {
	IsAvailable *bool `json:"is_available"`
}

func (h *CatalogHandler) getCatalog(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catalog.GetCatalog(r.Context())
	if err != nil {
		log.Printf("error obteniendo catálogo: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "No se pudo obtener el catálogo")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"categories": categories})
}

func (h *CatalogHandler) setProductAvailability(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de producto inválido")
		return
	}

	req, ok := decodeAvailability(w, r)
	if !ok {
		return
	}

	product, err := h.catalog.SetProductAvailability(r.Context(), id, *req.IsAvailable)
	if errors.Is(err, domain.ErrProductNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Producto no encontrado")
		return
	}
	if err != nil {
		log.Printf("error cambiando disponibilidad de producto %d: %v", id, err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "No se pudo actualizar el producto")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, product)
}

func (h *CatalogHandler) setModifierAvailability(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de modificador inválido")
		return
	}

	req, ok := decodeAvailability(w, r)
	if !ok {
		return
	}

	option, err := h.catalog.SetModifierAvailability(r.Context(), id, *req.IsAvailable)
	if errors.Is(err, domain.ErrModifierNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Modificador no encontrado")
		return
	}
	if err != nil {
		log.Printf("error cambiando disponibilidad de modificador %d: %v", id, err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "No se pudo actualizar el modificador")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, option)
}

func decodeAvailability(w http.ResponseWriter, r *http.Request) (availabilityRequest, bool) {
	var req availabilityRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil || req.IsAvailable == nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "Se requiere el campo booleano is_available")
		return req, false
	}
	return req, true
}
