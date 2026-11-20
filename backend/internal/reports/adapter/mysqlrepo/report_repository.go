package mysqlrepo

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"buffalo-pos/internal/reports/domain"
)

type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

func (r *ReportRepository) GetDailySummary(ctx context.Context, date string) (*domain.DailySummary, error) {
	summary := &domain.DailySummary{
		Date:        date,
		ByMethod:    []domain.PaymentMethodBreakdown{},
		TopProducts: []domain.TopProduct{},
	}

	// 1. Totales de ventas pagadas del día
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COALESCE(SUM(subtotal), 0),
		       COALESCE(SUM(tax_amount), 0),
		       COALESCE(SUM(tip_amount), 0),
		       COALESCE(SUM(total_amount), 0)
		FROM payments
		WHERE DATE(created_at) = ?
	`, date).Scan(
		&summary.PaidOrders,
		&summary.Subtotal,
		&summary.TaxTotal,
		&summary.TipTotal,
		&summary.TotalRevenue,
	)
	if err != nil {
		return nil, fmt.Errorf("consultando totales de pagos: %w", err)
	}

	// 2. Desglose por método de pago
	// 2. Desglose por método de pago
	methodRows, err := r.db.QueryContext(ctx, `
		SELECT payment_method, COUNT(*), COALESCE(SUM(total_amount), 0) AS total
		FROM payments
		WHERE DATE(created_at) = ?
		GROUP BY payment_method
		ORDER BY total DESC
	`, date)
	if err != nil {
		return nil, fmt.Errorf("consultando desglose por método: %w", err)
	}
	defer methodRows.Close()

	for methodRows.Next() {
		var m domain.PaymentMethodBreakdown
		if err := methodRows.Scan(&m.Method, &m.Count, &m.Total); err != nil {
			return nil, fmt.Errorf("escaneando método de pago: %w", err)
		}
		summary.ByMethod = append(summary.ByMethod, m)
	}
	if err := methodRows.Err(); err != nil {
		return nil, err
	}

	// 3. Top productos vendidos del día (ítems no anulados en órdenes pagadas)
	prodRows, err := r.db.QueryContext(ctx, `
		SELECT oi.product_id, oi.product_name,
		       COALESCE(SUM(oi.quantity), 0) AS units_sold,
		       COALESCE(SUM(oi.quantity * oi.unit_price), 0) AS revenue
		FROM order_items oi
		JOIN payments p ON p.order_id = oi.order_id
		WHERE DATE(p.created_at) = ? AND oi.status <> 'VOIDED'
		GROUP BY oi.product_id, oi.product_name
		ORDER BY units_sold DESC, revenue DESC
		LIMIT 10
	`, date)
	if err != nil {
		return nil, fmt.Errorf("consultando top productos: %w", err)
	}
	defer prodRows.Close()

	for prodRows.Next() {
		var tp domain.TopProduct
		if err := prodRows.Scan(&tp.ProductID, &tp.ProductName, &tp.UnitsSold, &tp.Revenue); err != nil {
			return nil, fmt.Errorf("escaneando top producto: %w", err)
		}
		summary.TopProducts = append(summary.TopProducts, tp)
	}
	if err := prodRows.Err(); err != nil {
		return nil, err
	}

	// 4. Resumen de anulaciones del día
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(quantity * unit_price), 0)
		FROM order_items
		WHERE DATE(created_at) = ? AND status = 'VOIDED'
	`, date).Scan(&summary.VoidedCount, &summary.VoidedAmount)
	if err != nil {
		return nil, fmt.Errorf("consultando resumen de anulaciones: %w", err)
	}

	// 5. Métricas de cocina (tiempo promedio en minutos hasta READY)
	var avgMins float64
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       COALESCE(AVG(TIMESTAMPDIFF(SECOND, created_at, ready_at)) / 60.0, 0)
		FROM order_items
		WHERE DATE(created_at) = ? AND ready_at IS NOT NULL AND status <> 'VOIDED'
	`, date).Scan(&summary.KitchenMetrics.ReadyItemsCount, &avgMins)
	if err != nil {
		return nil, fmt.Errorf("consultando métricas de cocina: %w", err)
	}
	summary.KitchenMetrics.AvgPrepMinutes = math.Round(avgMins*100) / 100

	return summary, nil
}

func (r *ReportRepository) ListVoids(ctx context.Context, date string) ([]domain.VoidDetail, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT oi.id, oi.order_id, t.code, oi.product_name, oi.quantity, oi.unit_price,
		       COALESCE(oi.void_reason, ''), COALESCE(u.employee_code, ''), COALESCE(u.full_name, ''),
		       oi.created_at
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN dining_tables t ON t.id = o.table_id
		LEFT JOIN users u ON u.id = oi.voided_by
		WHERE DATE(oi.created_at) = ? AND oi.status = 'VOIDED'
		ORDER BY oi.created_at DESC, oi.id DESC
	`, date)
	if err != nil {
		return nil, fmt.Errorf("consultando detalle de anulaciones: %w", err)
	}
	defer rows.Close()

	voids := []domain.VoidDetail{}
	for rows.Next() {
		var v domain.VoidDetail
		if err := rows.Scan(
			&v.ItemID, &v.OrderID, &v.TableCode, &v.ProductName, &v.Quantity, &v.UnitPrice,
			&v.VoidReason, &v.VoidedByCode, &v.VoidedByName, &v.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("escaneando anulación: %w", err)
		}
		voids = append(voids, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return voids, nil
}
