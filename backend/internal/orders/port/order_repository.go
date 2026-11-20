package port

import (
	"context"

	"buffalo-pos/internal/orders/domain"
)

type OrderRepository interface {
	CreateOrAppend(ctx context.Context, tableID, waiterID int64, items []domain.NewOrderItemInput) (*domain.Order, error)
	GetByID(ctx context.Context, orderID int64) (*domain.Order, error)
	GetOpenByTableID(ctx context.Context, tableID int64) (*domain.Order, error)
	VerifyManagerCredentials(ctx context.Context, employeeCode string) (userID int64, pinHash string, err error)
	VoidItem(ctx context.Context, orderID, itemID, managerID int64, reason string) (*domain.Order, error)
}
