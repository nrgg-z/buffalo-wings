package service

import (
	"context"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"buffalo-pos/internal/orders/domain"
	"buffalo-pos/internal/orders/port"
)

type OrderService struct {
	repo port.OrderRepository
}

func NewOrderService(repo port.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateOrAppend(ctx context.Context, tableID, waiterID int64, items []domain.NewOrderItemInput) (*domain.Order, error) {
	if len(items) == 0 {
		return nil, domain.ErrEmptyItems
	}
	for i := range items {
		if items[i].ProductID <= 0 {
			return nil, domain.ErrProductNotFound
		}
		if items[i].Quantity <= 0 {
			return nil, domain.ErrInvalidQuantity
		}
		items[i].Notes = strings.TrimSpace(items[i].Notes)
	}
	return s.repo.CreateOrAppend(ctx, tableID, waiterID, items)
}

func (s *OrderService) GetByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	return s.repo.GetByID(ctx, orderID)
}

func (s *OrderService) GetOpenByTableID(ctx context.Context, tableID int64) (*domain.Order, error) {
	return s.repo.GetOpenByTableID(ctx, tableID)
}

func (s *OrderService) VoidItem(ctx context.Context, orderID, itemID int64, managerCode, managerPIN, reason string) (*domain.Order, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, domain.ErrVoidReasonRequired
	}
	managerCode = strings.TrimSpace(managerCode)
	if managerCode == "" || managerPIN == "" {
		return nil, domain.ErrInvalidManagerPIN
	}

	managerID, pinHash, err := s.repo.VerifyManagerCredentials(ctx, managerCode)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pinHash), []byte(managerPIN)); err != nil {
		return nil, domain.ErrInvalidManagerPIN
	}

	return s.repo.VoidItem(ctx, orderID, itemID, managerID, reason)
}
