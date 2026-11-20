package port

import (
	"context"

	"buffalo-pos/internal/kds/domain"
)

type KDSRepository interface {
	ListQueue(ctx context.Context, station string) ([]domain.KDSItem, error)
	ListReady(ctx context.Context) ([]domain.KDSItem, error)
	UpdateItemStatus(ctx context.Context, itemID int64, nextStatus string) (*domain.KDSItem, error)
}
