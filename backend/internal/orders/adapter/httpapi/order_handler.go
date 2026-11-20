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
	"buffalo-pos/internal/orders/domain"
	"buffalo-pos/internal/platform/httpx"
)

const maxBodyBytes = 16 << 10

type OrderService interface {
	CreateOrAppend(ctx context.Context, tableID, waiterID int64, items []domain.NewOrderItemInput) (*domain.Order, error)
	GetByID(ctx context.Context, orderID int64) (*domain.Order, error)
	GetOpenByTableID(ctx context.Context, tableID int64) (*domain.Order, error)
	VoidItem(ctx context.Context, orderID, itemID int64, managerCode, managerPIN, reason string) (*domain.Order, error)
}

type OrderHandler struct {
	orders OrderService
}

func NewOrderHandler(orders OrderService) *OrderHandler {
	return &OrderHandler{orders: orders}
}

// Register registra las rutas de comandas. ADMIN siempre pasa.
func (h *OrderHandler) Register(mux *http.ServeMux, v authhttp.TokenVerifier) {
	take := authhttp.Protected(v, authdomain.RoleWaiter, authdomain.RoleManager)
	view := authhttp.Protected(v, authdomain.RoleWaiter, authdomain.RoleCashier, authdomain.RoleKitchen, authdomain.RoleHostess, authdomain.RoleManager)

	mux.Handle("POST /api/v1/orders", take(http.HandlerFunc(h.createOrAppend)))
	mux.Handle("GET /api/v1/orders/{id}", view(http.HandlerFunc(h.getByID)))
	mux.Handle("GET /api/v1/tables/{id}/order", view(http.HandlerFunc(h.getOpenByTable)))
	mux.Handle("POST /api/v1/orders/{id}/items/{itemId}/void", take(http.HandlerFunc(h.voidItem)))
}

type createOrderRequest struct {
	TableID int64                      `json:"table_id"`
	Items   []domain.NewOrderItemInput `json:"items"`
}

type voidItemRequest struct {
	ManagerCode string `json:"manager_code"`
	ManagerPIN  string `json:"manager_pin"`
	Reason      string `json:"reason"`
}

func (h *OrderHandler) createOrAppend(w http.ResponseWriter, r *http.Request) {
	p, ok := authhttp.PrincipalFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Sesión requerida")
		return
	}

	var req createOrderRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil || req.TableID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "Se requiere table_id y la lista de items")
		return
	}

	order, err := h.orders.CreateOrAppend(r.Context(), req.TableID, p.UserID, req.Items)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, order)
}

func (h *OrderHandler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de comanda inválido")
		return
	}
	order, err := h.orders.GetByID(r.Context(), id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) getOpenByTable(w http.ResponseWriter, r *http.Request) {
	tableID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || tableID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de mesa inválido")
		return
	}
	order, err := h.orders.GetOpenByTableID(r.Context(), tableID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) voidItem(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orderID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de comanda inválido")
		return
	}
	itemID, err := strconv.ParseInt(r.PathValue("itemId"), 10, 64)
	if err != nil || itemID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de ítem inválido")
		return
	}

	var req voidItemRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "Cuerpo de anulación inválido")
		return
	}

	order, err := h.orders.VoidItem(r.Context(), orderID, itemID, req.ManagerCode, req.ManagerPIN, req.Reason)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, order)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrTableNotFound),
		errors.Is(err, domain.ErrOrderNotFound),
		errors.Is(err, domain.ErrItemNotFound),
		errors.Is(err, domain.ErrProductNotFound),
		errors.Is(err, domain.ErrModifierNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domain.ErrTableNotOccupied),
		errors.Is(err, domain.ErrOrderClosed),
		errors.Is(err, domain.ErrItemAlreadyVoided),
		errors.Is(err, domain.ErrProductUnavailable),
		errors.Is(err, domain.ErrModifierUnavailable):
		httpx.WriteError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, domain.ErrEmptyItems),
		errors.Is(err, domain.ErrInvalidQuantity),
		errors.Is(err, domain.ErrModifierRule),
		errors.Is(err, domain.ErrVoidReasonRequired):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, domain.ErrInvalidManagerPIN):
		httpx.WriteError(w, http.StatusForbidden, "manager_auth_failed", err.Error())
	default:
		log.Printf("error en módulo de comandas: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Error procesando la comanda")
	}
}
