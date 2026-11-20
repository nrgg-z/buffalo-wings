package service

import (
	"context"
	"errors"
	"testing"

	"buffalo-pos/internal/billing/domain"
	"buffalo-pos/internal/billing/port"
)

type fakeBillingRepo struct {
	summary *port.OrderSummary
	err     error
}

func (f *fakeBillingRepo) GetOrderSummary(_ context.Context, _ int64) (*port.OrderSummary, error) {
	return f.summary, f.err
}

func (f *fakeBillingRepo) PayOrder(_ context.Context, in domain.PaymentInput, _ domain.Config) (*domain.Payment, error) {
	return &domain.Payment{OrderID: in.OrderID, PaymentMethod: in.PaymentMethod}, f.err
}

func (f *fakeBillingRepo) GetReceiptByOrderID(_ context.Context, orderID int64) (*domain.Payment, error) {
	return &domain.Payment{OrderID: orderID}, f.err
}

func TestGetPrecheckCalculations(t *testing.T) {
	repo := &fakeBillingRepo{
		summary: &port.OrderSummary{
			OrderID:   1,
			TableID:   1,
			TableCode: "M-01",
			Status:    "OPEN",
			Subtotal:  29.00,
		},
	}
	svc := NewBillingService(repo, domain.DefaultConfig())

	pre, err := svc.GetPrecheck(context.Background(), 1)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if pre.TaxAmount != 3.77 {
		t.Errorf("IVA = %.2f, esperaba 3.77", pre.TaxAmount)
	}
	if pre.SuggestedTipAmount != 2.90 {
		t.Errorf("propina = %.2f, esperaba 2.90", pre.SuggestedTipAmount)
	}
	if pre.TotalWithoutTip != 32.77 {
		t.Errorf("total sin propina = %.2f, esperaba 32.77", pre.TotalWithoutTip)
	}
	if pre.TotalWithSuggestedTip != 35.67 {
		t.Errorf("total con propina = %.2f, esperaba 35.67", pre.TotalWithSuggestedTip)
	}
}

func TestPayOrderValidations(t *testing.T) {
	svc := NewBillingService(&fakeBillingRepo{}, domain.DefaultConfig())

	_, err := svc.PayOrder(context.Background(), domain.PaymentInput{
		OrderID:       1,
		PaymentMethod: "CRYPTO",
	})
	if !errors.Is(err, domain.ErrInvalidPaymentMethod) {
		t.Fatalf("esperaba ErrInvalidPaymentMethod, obtuve %v", err)
	}

	_, err = svc.PayOrder(context.Background(), domain.PaymentInput{
		OrderID:       1,
		PaymentMethod: "cash",
		TipAmount:     -5,
	})
	if !errors.Is(err, domain.ErrInvalidTip) {
		t.Fatalf("esperaba ErrInvalidTip, obtuve %v", err)
	}
}
