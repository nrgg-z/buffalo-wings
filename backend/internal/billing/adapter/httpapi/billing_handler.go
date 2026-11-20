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
	"buffalo-pos/internal/billing/domain"
	"buffalo-pos/internal/platform/httpx"
)

const maxBodyBytes = 4 << 10

type BillingService interface {
	GetPrecheck(ctx context.Context, orderID int64) (*domain.Precheck, error)
	PayOrder(ctx context.Context, in domain.PaymentInput) (*domain.Payment, error)
	GetReceipt(ctx context.Context, orderID int64) (*domain.Payment, error)
}

type BillingHandler struct {
	billing BillingService
}

func NewBillingHandler(billing BillingService) *BillingHandler {
	return &BillingHandler{billing: billing}
}

// Register agrega las rutas de cobro y precuenta. ADMIN siempre pasa.
func (h *BillingHandler) Register(mux *http.ServeMux, v authhttp.TokenVerifier) {
	view := authhttp.Protected(v, authdomain.RoleWaiter, authdomain.RoleCashier, authdomain.RoleManager)
	cashier := authhttp.Protected(v, authdomain.RoleCashier, authdomain.RoleManager)

	mux.Handle("GET /api/v1/billing/orders/{id}/precheck", view(http.HandlerFunc(h.getPrecheck)))
	mux.Handle("POST /api/v1/billing/orders/{id}/pay", cashier(http.HandlerFunc(h.payOrder)))
	mux.Handle("GET /api/v1/billing/orders/{id}/receipt", view(http.HandlerFunc(h.getReceipt)))
}

type payOrderRequest struct {
	PaymentMethod  string  `json:"payment_method"`
	TipAmount      float64 `json:"tip_amount"`
	AmountReceived float64 `json:"amount_received"`
	Reference      string  `json:"reference"`
}

func (h *BillingHandler) getPrecheck(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orderID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de comanda inválido")
		return
	}
	pre, err := h.billing.GetPrecheck(r.Context(), orderID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, pre)
}

func (h *BillingHandler) payOrder(w http.ResponseWriter, r *http.Request) {
	p, ok := authhttp.PrincipalFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Sesión requerida")
		return
	}

	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orderID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de comanda inválido")
		return
	}

	var req payOrderRequest
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_body", "Cuerpo de pago inválido")
		return
	}

	receipt, err := h.billing.PayOrder(r.Context(), domain.PaymentInput{
		OrderID:        orderID,
		CashierID:      p.UserID,
		PaymentMethod:  req.PaymentMethod,
		TipAmount:      req.TipAmount,
		AmountReceived: req.AmountReceived,
		Reference:      req.Reference,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, receipt)
}

func (h *BillingHandler) getReceipt(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || orderID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "ID de comanda inválido")
		return
	}
	receipt, err := h.billing.GetReceipt(r.Context(), orderID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, receipt)
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound),
		errors.Is(err, domain.ErrPaymentNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domain.ErrOrderNotOpen),
		errors.Is(err, domain.ErrEmptyOrder):
		httpx.WriteError(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, domain.ErrInvalidPaymentMethod),
		errors.Is(err, domain.ErrInvalidTip),
		errors.Is(err, domain.ErrInsufficientAmount):
		httpx.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		log.Printf("error en módulo de cobro: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Error procesando el cobro")
	}
}
