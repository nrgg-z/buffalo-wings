package mysqlrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"buffalo-pos/internal/billing/domain"
	"buffalo-pos/internal/billing/port"
)

type BillingRepository struct {
	db *sql.DB
}

func NewBillingRepository(db *sql.DB) *BillingRepository {
	return &BillingRepository{db: db}
}

func (r *BillingRepository) GetOrderSummary(ctx context.Context, orderID int64) (*port.OrderSummary, error) {
	var s port.OrderSummary
	err := r.db.QueryRowContext(ctx, `
		SELECT o.id, o.table_id, t.code, o.status, o.subtotal
		FROM orders o
		JOIN dining_tables t ON t.id = o.table_id
		WHERE o.id = ?
	`, orderID).Scan(&s.OrderID, &s.TableID, &s.TableCode, &s.Status, &s.Subtotal)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultando comanda %d para precuenta: %w", orderID, err)
	}
	return &s, nil
}

func (r *BillingRepository) PayOrder(ctx context.Context, in domain.PaymentInput, cfg domain.Config) (*domain.Payment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciando transacción de cobro: %w", err)
	}
	defer tx.Rollback()

	var tableID int64
	var status string
	var subtotal float64
	err = tx.QueryRowContext(ctx, `
		SELECT table_id, status, subtotal
		FROM orders
		WHERE id = ?
		FOR UPDATE
	`, in.OrderID).Scan(&tableID, &status, &subtotal)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bloqueando comanda %d: %w", in.OrderID, err)
	}
	if status != "OPEN" {
		return nil, domain.ErrOrderNotOpen
	}
	if subtotal <= 0 {
		return nil, domain.ErrEmptyOrder
	}

	taxAmount := domain.Round2(subtotal * cfg.TaxRate)
	tipAmount := domain.Round2(in.TipAmount)
	totalAmount := domain.Round2(subtotal + taxAmount + tipAmount)

	amountReceived := domain.Round2(in.AmountReceived)
	if (in.PaymentMethod == "CARD" || in.PaymentMethod == "QR") && amountReceived == 0 {
		amountReceived = totalAmount
	}
	if amountReceived < totalAmount {
		return nil, domain.ErrInsufficientAmount
	}
	changeAmount := domain.Round2(amountReceived - totalAmount)
	dteCode := fmt.Sprintf("%s-%08d", cfg.DTEPrefix, in.OrderID)

	var ref any
	if in.Reference != "" {
		ref = in.Reference
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (
			order_id, cashier_id, payment_method, subtotal, tax_amount,
			tip_amount, total_amount, amount_received, change_amount, dte_code, reference
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, in.OrderID, in.CashierID, in.PaymentMethod, subtotal, taxAmount,
		tipAmount, totalAmount, amountReceived, changeAmount, dteCode, ref)
	if err != nil {
		return nil, fmt.Errorf("registrando pago de comanda %d: %w", in.OrderID, err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE orders SET status = 'PAID', closed_at = CURRENT_TIMESTAMP WHERE id = ?
	`, in.OrderID)
	if err != nil {
		return nil, fmt.Errorf("cerrando comanda %d: %w", in.OrderID, err)
	}

	// Pasa la mesa principal (y cualquier mesa unida a ella) a CLEANING
	_, err = tx.ExecContext(ctx, `
		UPDATE dining_tables
		SET status = 'CLEANING',
		    current_guests = 0,
		    assigned_waiter_id = NULL,
		    merged_into_id = NULL,
		    version = version + 1
		WHERE id = ? OR merged_into_id = ?
	`, tableID, tableID)
	if err != nil {
		return nil, fmt.Errorf("actualizando mesa %d a CLEANING: %w", tableID, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmando cobro de comanda %d: %w", in.OrderID, err)
	}

	return r.GetReceiptByOrderID(ctx, in.OrderID)
}

func (r *BillingRepository) GetReceiptByOrderID(ctx context.Context, orderID int64) (*domain.Payment, error) {
	var p domain.Payment
	var ref sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT pay.id, pay.order_id, o.table_id, t.code, pay.cashier_id,
		       pay.payment_method, pay.subtotal, pay.tax_amount, pay.tip_amount,
		       pay.total_amount, pay.amount_received, pay.change_amount,
		       pay.dte_code, pay.reference, pay.created_at
		FROM payments pay
		JOIN orders o ON o.id = pay.order_id
		JOIN dining_tables t ON t.id = o.table_id
		WHERE pay.order_id = ?
	`, orderID).Scan(
		&p.ID, &p.OrderID, &p.TableID, &p.TableCode, &p.CashierID,
		&p.PaymentMethod, &p.Subtotal, &p.TaxAmount, &p.TipAmount,
		&p.TotalAmount, &p.AmountReceived, &p.ChangeAmount,
		&p.DTECode, &ref, &p.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPaymentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultando recibo de comanda %d: %w", orderID, err)
	}
	if ref.Valid {
		p.Reference = ref.String
	}
	return &p, nil
}
