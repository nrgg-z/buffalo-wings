package service

import (
	"context"
	"strings"
	"time"

	"buffalo-pos/internal/kds/domain"
	"buffalo-pos/internal/kds/port"
)

type KDSService struct {
	repo port.KDSRepository
	now  func() time.Time
}

func NewKDSService(repo port.KDSRepository) *KDSService {
	return &KDSService{repo: repo, now: time.Now}
}

func (s *KDSService) ListQueue(ctx context.Context, station string) ([]domain.KDSItem, error) {
	station = strings.ToUpper(strings.TrimSpace(station))
	if station != "" && station != "KITCHEN" && station != "BAR" {
		return nil, domain.ErrInvalidStation
	}
	items, err := s.repo.ListQueue(ctx, station)
	if err != nil {
		return nil, err
	}
	now := s.now()
	for i := range items {
		items[i].ComputeTiming(now)
	}
	return items, nil
}

func (s *KDSService) ListReady(ctx context.Context) ([]domain.KDSItem, error) {
	items, err := s.repo.ListReady(ctx)
	if err != nil {
		return nil, err
	}
	now := s.now()
	for i := range items {
		items[i].ComputeTiming(now)
	}
	return items, nil
}

func (s *KDSService) UpdateItemStatus(ctx context.Context, itemID int64, nextStatus string) (*domain.KDSItem, error) {
	nextStatus = strings.ToUpper(strings.TrimSpace(nextStatus))
	if nextStatus != "IN_PROGRESS" && nextStatus != "READY" && nextStatus != "DELIVERED" {
		return nil, domain.ErrInvalidStatus
	}
	item, err := s.repo.UpdateItemStatus(ctx, itemID, nextStatus)
	if err != nil {
		return nil, err
	}
	item.ComputeTiming(s.now())
	return item, nil
}
