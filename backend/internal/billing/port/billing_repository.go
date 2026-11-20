package port

import (
	"context"

	"buffalo-pos/internal/billing/domain"
)

type OrderSummary struct {
	OrderID   int64
	TableID   int64
	TableCode string
	Status    string
	Subtotal  float64
}

type BillingRepository interface {
	GetOrderSummary(ctx context.Context, orderID int64) (*OrderSummary, error)
	PayOrder(ctx context.Context, in domain.PaymentInput, cfg domain.Config) (*domain.Payment, error)
	GetReceiptByOrderID(ctx context.Context, orderID int64) (*domain.Payment, error)
}
