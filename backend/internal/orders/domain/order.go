package domain

import (
	"errors"
	"time"
)

var (
	ErrTableNotFound       = errors.New("mesa no encontrada")
	ErrTableNotOccupied    = errors.New("la mesa no está ocupada")
	ErrOrderNotFound       = errors.New("comanda no encontrada")
	ErrOrderClosed         = errors.New("la comanda ya está cerrada")
	ErrItemNotFound        = errors.New("ítem no encontrado en la comanda")
	ErrItemAlreadyVoided   = errors.New("el ítem ya fue anulado")
	ErrEmptyItems          = errors.New("debe incluir al menos un producto")
	ErrInvalidQuantity     = errors.New("la cantidad debe ser mayor a cero")
	ErrProductNotFound     = errors.New("producto no encontrado")
	ErrProductUnavailable  = errors.New("producto agotado")
	ErrModifierNotFound    = errors.New("modificador no encontrado")
	ErrModifierUnavailable = errors.New("modificador agotado")
	ErrModifierRule        = errors.New("selección de modificadores inválida")
	ErrInvalidManagerPIN   = errors.New("autorización de gerente inválida")
	ErrVoidReasonRequired  = errors.New("el motivo de anulación es obligatorio")
)

type OrderItemModifier struct {
	ID               int64   `json:"id"`
	ModifierOptionID int64   `json:"modifier_option_id"`
	Name             string  `json:"name"`
	ExtraPrice       float64 `json:"extra_price"`
}

type OrderItem struct {
	ID          int64               `json:"id"`
	OrderID     int64               `json:"order_id"`
	ProductID   int64               `json:"product_id"`
	ProductName string              `json:"product_name"`
	Station     string              `json:"station"`
	Quantity    int                 `json:"quantity"`
	UnitPrice   float64             `json:"unit_price"`
	Notes       string              `json:"notes,omitempty"`
	Status      string              `json:"status"`
	VoidReason  string              `json:"void_reason,omitempty"`
	VoidedBy    *int64              `json:"voided_by,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	ReadyAt     *time.Time          `json:"ready_at,omitempty"`
	Modifiers   []OrderItemModifier `json:"modifiers"`
}

type Order struct {
	ID        int64       `json:"id"`
	TableID   int64       `json:"table_id"`
	TableCode string      `json:"table_code"`
	WaiterID  int64       `json:"waiter_id"`
	Status    string      `json:"status"`
	Subtotal  float64     `json:"subtotal"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	Items     []OrderItem `json:"items"`
}

type NewOrderItemInput struct {
	ProductID         int64   `json:"product_id"`
	Quantity          int     `json:"quantity"`
	Notes             string  `json:"notes"`
	ModifierOptionIDs []int64 `json:"modifier_option_ids"`
}
