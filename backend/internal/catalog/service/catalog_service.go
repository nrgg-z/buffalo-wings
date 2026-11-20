package service

import (
	"context"

	"buffalo-pos/internal/catalog/domain"
	"buffalo-pos/internal/catalog/port"
)

type CatalogService struct {
	repo port.CatalogRepository
}

func NewCatalogService(repo port.CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

func (s *CatalogService) GetCatalog(ctx context.Context) ([]domain.Category, error) {
	return s.repo.GetFullCatalog(ctx)
}

func (s *CatalogService) SetProductAvailability(ctx context.Context, productID int64, available bool) (*domain.Product, error) {
	return s.repo.SetProductAvailability(ctx, productID, available)
}

func (s *CatalogService) SetModifierAvailability(ctx context.Context, optionID int64, available bool) (*domain.ModifierOption, error) {
	return s.repo.SetModifierAvailability(ctx, optionID, available)
}
