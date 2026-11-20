package domain

import (
	"errors"
	"time"
)

var ErrInvalidDate = errors.New("formato de fecha inválido (usa YYYY-MM-DD)")

type PaymentMethodBreakdown struct {
	Method string  `json:"method"`
	Count  int     `json:"count"`
	Total  float64 `json:"total"`
}

type TopProduct struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitsSold   int     `json:"units_sold"`
	Revenue     float64 `json:"revenue"`
}

type KitchenMetrics struct {
	ReadyItemsCount int     `json:"ready_items_count"`
	AvgPrepMinutes  float64 `json:"avg_prep_minutes"`
}

type DailySummary struct {
	Date           string                   `json:"date"`
	PaidOrders     int                      `json:"paid_orders"`
	Subtotal       float64                  `json:"subtotal"`
	TaxTotal       float64                  `json:"tax_total"`
	TipTotal       float64                  `json:"tip_total"`
	TotalRevenue   float64                  `json:"total_revenue"`
	VoidedCount    int                      `json:"voided_count"`
	VoidedAmount   float64                  `json:"voided_amount"`
	ByMethod       []PaymentMethodBreakdown `json:"by_payment_method"`
	TopProducts    []TopProduct             `json:"top_products"`
	KitchenMetrics KitchenMetrics           `json:"kitchen_metrics"`
}

type VoidDetail struct {
	ItemID       int64     `json:"item_id"`
	OrderID      int64     `json:"order_id"`
	TableCode    string    `json:"table_code"`
	ProductName  string    `json:"product_name"`
	Quantity     int       `json:"quantity"`
	UnitPrice    float64   `json:"unit_price"`
	VoidReason   string    `json:"void_reason"`
	VoidedByCode string    `json:"voided_by_code"`
	VoidedByName string    `json:"voided_by_name"`
	CreatedAt    time.Time `json:"created_at"`
}
