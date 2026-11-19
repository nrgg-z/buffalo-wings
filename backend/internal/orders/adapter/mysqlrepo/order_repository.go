package mysqlrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"buffalo-pos/internal/orders/domain"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

type groupRule struct {
	required bool
	minSel   int
	maxSel   int
}

type selectedMod struct {
	optionID   int64
	name       string
	extraPrice float64
}

func (r *OrderRepository) CreateOrAppend(ctx context.Context, tableID, waiterID int64, items []domain.NewOrderItemInput) (*domain.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciando transacción de comanda: %w", err)
	}
	defer tx.Rollback()

	// 1. Validar que la mesa exista y esté OCUPADA
	var tableStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM dining_tables WHERE id = ? AND is_active = 1 FOR UPDATE
	`, tableID).Scan(&tableStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrTableNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultando mesa %d: %w", tableID, err)
	}
	if tableStatus != "OCCUPIED" {
		return nil, domain.ErrTableNotOccupied
	}

	// 2. Buscar comanda abierta de la mesa o crear una nueva
	var orderID int64
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM orders WHERE table_id = ? AND status = 'OPEN' ORDER BY id DESC LIMIT 1 FOR UPDATE
	`, tableID).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		res, insErr := tx.ExecContext(ctx, `
			INSERT INTO orders (table_id, waiter_id, status, subtotal) VALUES (?, ?, 'OPEN', 0.00)
		`, tableID, waiterID)
		if insErr != nil {
			return nil, fmt.Errorf("creando comanda: %w", insErr)
		}
		orderID, err = res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("obteniendo id de comanda: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("buscando comanda abierta: %w", err)
	}

	// 3. Validar e insertar cada ítem con sus modificadores
	for _, it := range items {
		var prodName, station string
		var basePrice float64
		var prodAvailable bool

		err = tx.QueryRowContext(ctx, `
			SELECT p.name, p.base_price, p.is_available, c.station
			FROM products p
			JOIN categories c ON c.id = p.category_id
			WHERE p.id = ? AND p.is_active = 1
		`, it.ProductID).Scan(&prodName, &basePrice, &prodAvailable, &station)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("consultando producto %d: %w", it.ProductID, err)
		}
		if !prodAvailable {
			return nil, domain.ErrProductUnavailable
		}

		// Reglas de grupos de modificadores asignados al producto
		ruleRows, err := tx.QueryContext(ctx, `
			SELECT mg.id, mg.is_required, mg.min_selections, mg.max_selections
			FROM product_modifier_groups pmg
			JOIN modifier_groups mg ON mg.id = pmg.group_id
			WHERE pmg.product_id = ? AND mg.is_active = 1
		`, it.ProductID)
		if err != nil {
			return nil, fmt.Errorf("consultando reglas de modificadores: %w", err)
		}
		rules := map[int64]groupRule{}
		for ruleRows.Next() {
			var gid int64
			var gr groupRule
			if err := ruleRows.Scan(&gid, &gr.required, &gr.minSel, &gr.maxSel); err != nil {
				ruleRows.Close()
				return nil, fmt.Errorf("escaneando regla de modificador: %w", err)
			}
			rules[gid] = gr
		}
		ruleRows.Close()
		if err := ruleRows.Err(); err != nil {
			return nil, err
		}

		// Validar modificadores seleccionados
		countsByGroup := map[int64]int{}
		seenOpt := map[int64]bool{}
		selected := make([]selectedMod, 0, len(it.ModifierOptionIDs))

		for _, optID := range it.ModifierOptionIDs {
			if seenOpt[optID] {
				continue
			}
			seenOpt[optID] = true

			var m selectedMod
			var gid int64
			var optAvailable bool
			err = tx.QueryRowContext(ctx, `
				SELECT id, group_id, name, extra_price, is_available
				FROM modifier_options
				WHERE id = ? AND is_active = 1
			`, optID).Scan(&m.optionID, &gid, &m.name, &m.extraPrice, &optAvailable)
			if errors.Is(err, sql.ErrNoRows) {
				return nil, domain.ErrModifierNotFound
			}
			if err != nil {
				return nil, fmt.Errorf("consultando modificador %d: %w", optID, err)
			}
			if !optAvailable {
				return nil, domain.ErrModifierUnavailable
			}
			if _, allowed := rules[gid]; !allowed {
				return nil, domain.ErrModifierRule
			}
			countsByGroup[gid]++
			selected = append(selected, m)
		}

		// Verificar mínimos y máximos por grupo
		for gid, gr := range rules {
			cnt := countsByGroup[gid]
			if gr.required && cnt < gr.minSel {
				return nil, domain.ErrModifierRule
			}
			if cnt > 0 && cnt < gr.minSel {
				return nil, domain.ErrModifierRule
			}
			if gr.maxSel > 0 && cnt > gr.maxSel {
				return nil, domain.ErrModifierRule
			}
		}

		var notes any
		if it.Notes != "" {
			notes = it.Notes
		}

		itemRes, err := tx.ExecContext(ctx, `
			INSERT INTO order_items (order_id, product_id, product_name, station, quantity, unit_price, notes, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING')
		`, orderID, it.ProductID, prodName, station, it.Quantity, basePrice, notes)
		if err != nil {
			return nil, fmt.Errorf("insertando ítem de comanda: %w", err)
		}
		itemID, err := itemRes.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("obteniendo id de ítem: %w", err)
		}

		for _, m := range selected {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO order_item_modifiers (order_item_id, modifier_option_id, modifier_name, extra_price)
				VALUES (?, ?, ?, ?)
			`, itemID, m.optionID, m.name, m.extraPrice)
			if err != nil {
				return nil, fmt.Errorf("insertando modificador de ítem: %w", err)
			}
		}
	}

	if err := recalcSubtotalTx(ctx, tx, orderID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmando comanda: %w", err)
	}

	return r.GetByID(ctx, orderID)
}

func (r *OrderRepository) GetByID(ctx context.Context, orderID int64) (*domain.Order, error) {
	var o domain.Order
	err := r.db.QueryRowContext(ctx, `
		SELECT o.id, o.table_id, t.code, o.waiter_id, o.status, o.subtotal, o.created_at, o.updated_at
		FROM orders o
		JOIN dining_tables t ON t.id = o.table_id
		WHERE o.id = ?
	`, orderID).Scan(&o.ID, &o.TableID, &o.TableCode, &o.WaiterID, &o.Status, &o.Subtotal, &o.CreatedAt, &o.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultando comanda %d: %w", orderID, err)
	}

	itemRows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, product_id, product_name, station, quantity, unit_price, notes, status, void_reason, voided_by, created_at, ready_at
		FROM order_items
		WHERE order_id = ?
		ORDER BY id ASC
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("consultando ítems de comanda %d: %w", orderID, err)
	}
	defer itemRows.Close()

	o.Items = []domain.OrderItem{}
	itemIdx := map[int64]int{}
	for itemRows.Next() {
		var it domain.OrderItem
		var notes, voidReason sql.NullString
		var voidedBy sql.NullInt64
		var readyAt sql.NullTime

		if err := itemRows.Scan(
			&it.ID, &it.OrderID, &it.ProductID, &it.ProductName, &it.Station,
			&it.Quantity, &it.UnitPrice, &notes, &it.Status, &voidReason, &voidedBy,
			&it.CreatedAt, &readyAt,
		); err != nil {
			return nil, fmt.Errorf("escaneando ítem de comanda: %w", err)
		}
		if notes.Valid {
			it.Notes = notes.String
		}
		if voidReason.Valid {
			it.VoidReason = voidReason.String
		}
		if voidedBy.Valid {
			v := voidedBy.Int64
			it.VoidedBy = &v
		}
		if readyAt.Valid {
			t := readyAt.Time
			it.ReadyAt = &t
		}
		it.Modifiers = []domain.OrderItemModifier{}
		itemIdx[it.ID] = len(o.Items)
		o.Items = append(o.Items, it)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	modRows, err := r.db.QueryContext(ctx, `
		SELECT oim.id, oim.order_item_id, oim.modifier_option_id, oim.modifier_name, oim.extra_price
		FROM order_item_modifiers oim
		JOIN order_items oi ON oi.id = oim.order_item_id
		WHERE oi.order_id = ?
		ORDER BY oim.id ASC
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("consultando modificadores de comanda %d: %w", orderID, err)
	}
	defer modRows.Close()

	for modRows.Next() {
		var m domain.OrderItemModifier
		var itemID int64
		if err := modRows.Scan(&m.ID, &itemID, &m.ModifierOptionID, &m.Name, &m.ExtraPrice); err != nil {
			return nil, fmt.Errorf("escaneando modificador de comanda: %w", err)
		}
		if idx, ok := itemIdx[itemID]; ok {
			o.Items[idx].Modifiers = append(o.Items[idx].Modifiers, m)
		}
	}
	if err := modRows.Err(); err != nil {
		return nil, err
	}

	return &o, nil
}

func (r *OrderRepository) GetOpenByTableID(ctx context.Context, tableID int64) (*domain.Order, error) {
	var orderID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM orders WHERE table_id = ? AND status = 'OPEN' ORDER BY id DESC LIMIT 1
	`, tableID).Scan(&orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("buscando comanda abierta de mesa %d: %w", tableID, err)
	}
	return r.GetByID(ctx, orderID)
}

func (r *OrderRepository) VerifyManagerCredentials(ctx context.Context, employeeCode string) (int64, string, error) {
	var id int64
	var hash string
	err := r.db.QueryRowContext(ctx, `
		SELECT u.id, u.pin_hash
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.employee_code = ? AND u.is_active = 1 AND r.code IN ('ADMIN', 'MANAGER')
	`, employeeCode).Scan(&id, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", domain.ErrInvalidManagerPIN
	}
	if err != nil {
		return 0, "", fmt.Errorf("verificando credenciales de gerente: %w", err)
	}
	return id, hash, nil
}

func (r *OrderRepository) VoidItem(ctx context.Context, orderID, itemID, managerID int64, reason string) (*domain.Order, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("iniciando transacción de anulación: %w", err)
	}
	defer tx.Rollback()

	var orderStatus string
	err = tx.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ? FOR UPDATE`, orderID).Scan(&orderStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultando comanda %d: %w", orderID, err)
	}
	if orderStatus != "OPEN" {
		return nil, domain.ErrOrderClosed
	}

	var itemStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status FROM order_items WHERE id = ? AND order_id = ? FOR UPDATE
	`, itemID, orderID).Scan(&itemStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consultando ítem %d: %w", itemID, err)
	}
	if itemStatus == "VOIDED" {
		return nil, domain.ErrItemAlreadyVoided
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE order_items SET status = 'VOIDED', void_reason = ?, voided_by = ? WHERE id = ?
	`, reason, managerID, itemID)
	if err != nil {
		return nil, fmt.Errorf("anulando ítem %d: %w", itemID, err)
	}

	if err := recalcSubtotalTx(ctx, tx, orderID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("confirmando anulación: %w", err)
	}

	return r.GetByID(ctx, orderID)
}

func recalcSubtotalTx(ctx context.Context, tx *sql.Tx, orderID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE orders
		SET subtotal = COALESCE((
			SELECT SUM(
				oi.quantity * (
					oi.unit_price + COALESCE((
						SELECT SUM(oim.extra_price)
						FROM order_item_modifiers oim
						WHERE oim.order_item_id = oi.id
					), 0)
				)
			)
			FROM order_items oi
			WHERE oi.order_id = ? AND oi.status <> 'VOIDED'
		), 0.00)
		WHERE id = ?
	`, orderID, orderID)
	if err != nil {
		return fmt.Errorf("recalculando subtotal de comanda %d: %w", orderID, err)
	}
	return nil
}
