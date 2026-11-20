package domain

import (
	"errors"
	"math"
	"time"
)

var (
	ErrOrderNotFound        = errors.New("comanda no encontrada")
	ErrOrderNotOpen         = errors.New("la comanda ya está cerrada o pagada")
	ErrEmptyOrder           = errors.New("la comanda no tiene monto por cobrar")
	ErrPaymentNotFound      = errors.New("recibo de pago no encontrado")
	ErrInvalidPaymentMethod = errors.New("método de pago inválido (usa CASH, CARD o QR)")
	ErrInvalidTip           = errors.New("la propina no puede ser negativa")
	ErrInsufficientAmount   = errors.New("el monto recibido es menor al total a pagar")
)

type Config struct {
	TaxRate          float64
	SuggestedTipRate float64
	DTEPrefix        string
	EnabledMethods   []string
}

func DefaultConfig() Config {
	return Config{
		TaxRate:          0.13,
		SuggestedTipRate: 0.10,
		DTEPrefix:        "DTE-01-S001P001",
		EnabledMethods:   []string{"CASH", "CARD", "QR"},
	}
}

func (c Config) IsMethodEnabled(method string) bool {
	for _, m := range c.EnabledMethods {
		if m == method {
			return true
		}
	}
	return false
}

func Round2(v float64) float64 {
	return math.Round(v*100) / 100
}

type Precheck struct {
	OrderID               int64    `json:"order_id"`
	TableID               int64    `json:"table_id"`
	TableCode             string   `json:"table_code"`
	Status                string   `json:"status"`
	Subtotal              float64  `json:"subtotal"`
	TaxRate               float64  `json:"tax_rate"`
	TaxAmount             float64  `json:"tax_amount"`
	SuggestedTipRate      float64  `json:"suggested_tip_rate"`
	SuggestedTipAmount    float64  `json:"suggested_tip_amount"`
	TotalWithoutTip       float64  `json:"total_without_tip"`
	TotalWithSuggestedTip float64  `json:"total_with_suggested_tip"`
	EnabledPaymentMethods []string `json:"enabled_payment_methods"`
}

type PaymentInput struct {
	OrderID        int64
	CashierID      int64
	PaymentMethod  string
	TipAmount      float64
	AmountReceived float64
	Reference      string
}

type Payment struct {
	ID             int64     `json:"id"`
	OrderID        int64     `json:"order_id"`
	TableID        int64     `json:"table_id"`
	TableCode      string    `json:"table_code"`
	CashierID      int64     `json:"cashier_id"`
	PaymentMethod  string    `json:"payment_method"`
	Subtotal       float64   `json:"subtotal"`
	TaxAmount      float64   `json:"tax_amount"`
	TipAmount      float64   `json:"tip_amount"`
	TotalAmount    float64   `json:"total_amount"`
	AmountReceived float64   `json:"amount_received"`
	ChangeAmount   float64   `json:"change_amount"`
	DTECode        string    `json:"dte_code"`
	Reference      string    `json:"reference,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
