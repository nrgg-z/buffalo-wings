package port

import (
	"context"

	"buffalo-pos/internal/catalog/domain"
)

type CatalogRepository interface {
	GetFullCatalog(ctx context.Context) ([]domain.Category, error)
	SetProductAvailability(ctx context.Context, productID int64, available bool) (*domain.Product, error)
	SetModifierAvailability(ctx context.Context, optionID int64, available bool) (*domain.ModifierOption, error)
}
