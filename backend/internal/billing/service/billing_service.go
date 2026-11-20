package service

import (
	"context"
	"strings"

	"buffalo-pos/internal/billing/domain"
	"buffalo-pos/internal/billing/port"
)

type BillingService struct {
	repo port.BillingRepository
	cfg  domain.Config
}

func NewBillingService(repo port.BillingRepository, cfg domain.Config) *BillingService {
	def := domain.DefaultConfig()
	if cfg.TaxRate <= 0 {
		cfg.TaxRate = def.TaxRate
	}
	if cfg.SuggestedTipRate < 0 {
		cfg.SuggestedTipRate = def.SuggestedTipRate
	}
	if strings.TrimSpace(cfg.DTEPrefix) == "" {
		cfg.DTEPrefix = def.DTEPrefix
	}
	if len(cfg.EnabledMethods) == 0 {
		cfg.EnabledMethods = def.EnabledMethods
	}
	return &BillingService{repo: repo, cfg: cfg}
}

func (s *BillingService) GetPrecheck(ctx context.Context, orderID int64) (*domain.Precheck, error) {
	summary, err := s.repo.GetOrderSummary(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if summary.Status != "OPEN" {
		return nil, domain.ErrOrderNotOpen
	}
	if summary.Subtotal <= 0 {
		return nil, domain.ErrEmptyOrder
	}

	tax := domain.Round2(summary.Subtotal * s.cfg.TaxRate)
	tip := domain.Round2(summary.Subtotal * s.cfg.SuggestedTipRate)
	totalNoTip := domain.Round2(summary.Subtotal + tax)
	totalWithTip := domain.Round2(totalNoTip + tip)

	return &domain.Precheck{
		OrderID:               summary.OrderID,
		TableID:               summary.TableID,
		TableCode:             summary.TableCode,
		Status:                summary.Status,
		Subtotal:              summary.Subtotal,
		TaxRate:               s.cfg.TaxRate,
		TaxAmount:             tax,
		SuggestedTipRate:      s.cfg.SuggestedTipRate,
		SuggestedTipAmount:    tip,
		TotalWithoutTip:       totalNoTip,
		TotalWithSuggestedTip: totalWithTip,
		EnabledPaymentMethods: s.cfg.EnabledMethods,
	}, nil
}

func (s *BillingService) PayOrder(ctx context.Context, in domain.PaymentInput) (*domain.Payment, error) {
	in.PaymentMethod = strings.ToUpper(strings.TrimSpace(in.PaymentMethod))
	in.Reference = strings.TrimSpace(in.Reference)

	if !s.cfg.IsMethodEnabled(in.PaymentMethod) {
		return nil, domain.ErrInvalidPaymentMethod
	}
	if in.TipAmount < 0 {
		return nil, domain.ErrInvalidTip
	}
	return s.repo.PayOrder(ctx, in, s.cfg)
}

func (s *BillingService) GetReceipt(ctx context.Context, orderID int64) (*domain.Payment, error) {
	return s.repo.GetReceiptByOrderID(ctx, orderID)
}
