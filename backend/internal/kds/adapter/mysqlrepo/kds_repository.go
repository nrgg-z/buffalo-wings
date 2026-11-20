package mysqlrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"buffalo-pos/internal/kds/domain"
)

type KDSRepository struct {
	db *sql.DB
}

func NewKDSRepository(db *sql.DB) *KDSRepository {
	return &KDSRepository{db: db}
}

func (r *KDSRepository) ListQueue(ctx context.Context, station string) ([]domain.KDSItem, error) {
	query := `
		SELECT oi.id, oi.order_id, o.table_id, t.code, oi.product_name, oi.station,
		       oi.quantity, oi.notes, oi.status, oi.created_at, oi.ready_at
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN dining_tables t ON t.id = o.table_id
		WHERE o.status = 'OPEN' AND oi.status IN ('PENDING', 'IN_PROGRESS')
	`
	args := []any{}
	if station != "" {
		query += " AND oi.station = ?"
		args = append(args, station)
	}
	query += " ORDER BY oi.created_at ASC, oi.id ASC"

	return r.queryItems(ctx, query, args...)
}

func (r *KDSRepository) ListReady(ctx context.Context) ([]domain.KDSItem, error) {
	query := `
		SELECT oi.id, oi.order_id, o.table_id, t.code, oi.product_name, oi.station,
		       oi.quantity, oi.notes, oi.status, oi.created_at, oi.ready_at
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN dining_tables t ON t.id = o.table_id
		WHERE o.status = 'OPEN' AND oi.status = 'READY'
		ORDER BY oi.ready_at ASC, oi.id ASC
	`
	return r.queryItems(ctx, query)
}

func (r *KDSRepository) UpdateItemStatus(ctx context.Context, itemID int64, nextStatus string) (*domain.KDSItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciando transacción KDS: %w", err)
	}
	defer tx.Rollback()

	var currentStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT oi.status
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.id = ? AND o.status = 'OPEN'
		FOR UPDATE
	`, itemID).Scan(&currentStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultando estado de ítem %d: %w", itemID, err)
	}

	if !domain.IsValidTransition(currentStatus, nextStatus) {
		return nil, domain.ErrInvalidTransition
	}

	if nextStatus == "READY" {
		_, err = tx.ExecContext(ctx, `
			UPDATE order_items SET status = 'READY', ready_at = CURRENT_TIMESTAMP WHERE id = ?
		`, itemID)
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE order_items SET status = ? WHERE id = ?
		`, nextStatus, itemID)
	}
	if err != nil {
		return nil, fmt.Errorf("actualizando estado de ítem %d: %w", itemID, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmando cambio KDS: %w", err)
	}

	items, err := r.queryItems(ctx, `
		SELECT oi.id, oi.order_id, o.table_id, t.code, oi.product_name, oi.station,
		       oi.quantity, oi.notes, oi.status, oi.created_at, oi.ready_at
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		JOIN dining_tables t ON t.id = o.table_id
		WHERE oi.id = ?
	`, itemID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, domain.ErrItemNotFound
	}
	return &items[0], nil
}

func (r *KDSRepository) queryItems(ctx context.Context, query string, args ...any) ([]domain.KDSItem, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("consultando ítems KDS: %w", err)
	}
	defer rows.Close()

	items := []domain.KDSItem{}
	idxByID := map[int64]int{}
	for rows.Next() {
		var it domain.KDSItem
		var notes sql.NullString
		var readyAt sql.NullTime
		if err := rows.Scan(
			&it.ID, &it.OrderID, &it.TableID, &it.TableCode, &it.ProductName,
			&it.Station, &it.Quantity, &notes, &it.Status, &it.CreatedAt, &readyAt,
		); err != nil {
			return nil, fmt.Errorf("escaneando ítem KDS: %w", err)
		}
		if notes.Valid {
			it.Notes = notes.String
		}
		if readyAt.Valid {
			t := readyAt.Time
			it.ReadyAt = &t
		}
		it.Modifiers = []string{}
		idxByID[it.ID] = len(items)
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}

	modRows, err := r.db.QueryContext(ctx, `
		SELECT oim.order_item_id, oim.modifier_name
		FROM order_item_modifiers oim
		JOIN order_items oi ON oi.id = oim.order_item_id
		JOIN orders o ON o.id = oi.order_id
		WHERE o.status = 'OPEN'
		ORDER BY oim.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("consultando modificadores KDS: %w", err)
	}
	defer modRows.Close()

	for modRows.Next() {
		var itemID int64
		var name string
		if err := modRows.Scan(&itemID, &name); err != nil {
			return nil, fmt.Errorf("escaneando modificador KDS: %w", err)
		}
		if idx, ok := idxByID[itemID]; ok {
			items[idx].Modifiers = append(items[idx].Modifiers, name)
		}
	}
	if err := modRows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
