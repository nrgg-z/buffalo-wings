package mysqlrepo

import (
	"context"
	"database/sql"
	"fmt"

	"buffalo-pos/internal/tables/domain"
	"buffalo-pos/internal/tables/port"
)

// TableRepository implementa port.TableRepository usando MySQL.
type TableRepository struct {
	db *sql.DB
}

func NewTableRepository(db *sql.DB) *TableRepository {
	return &TableRepository{db: db}
}

// Verificación en compilación: si falta un método del contrato, no compila.
var _ port.TableRepository = (*TableRepository)(nil)

func (r *TableRepository) List(ctx context.Context) ([]domain.Table, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, code, zone, capacity, status, current_guests,
		       assigned_waiter_id, merged_into_id, version
		FROM dining_tables
		WHERE is_active = 1
		ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listar mesas: %w", err)
	}
	defer rows.Close()

	var tables []domain.Table
	for rows.Next() {
		var (
			t              domain.Table
			status         string
			waiter, merged sql.NullInt64
		)
		if err := rows.Scan(&t.ID, &t.Code, &t.Zone, &t.Capacity, &status,
			&t.CurrentGuests, &waiter, &merged, &t.Version); err != nil {
			return nil, fmt.Errorf("leer mesa: %w", err)
		}
		t.Status = domain.Status(status)
		if waiter.Valid {
			t.AssignedWaiterID = &waiter.Int64
		}
		if merged.Valid {
			t.MergedIntoID = &merged.Int64
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listar mesas: %w", err)
	}
	return tables, nil
}

func (r *TableRepository) Assign(ctx context.Context, ref port.TableRef, guests int) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE dining_tables
		SET status = 'OCCUPIED', current_guests = ?, version = version + 1
		WHERE id = ? AND version = ?`, guests, ref.ID, ref.Version)
	if err != nil {
		return fmt.Errorf("asignar mesa: %w", err)
	}
	return requireOneRow(res)
}

func (r *TableRepository) Merge(ctx context.Context, primary port.TableRef, others []port.TableRef) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("combinar mesas: %w", err)
	}
	defer tx.Rollback() // no hace nada si ya se hizo Commit

	// La principal solo sube de versión: así otra operación concurrente sobre ella falla.
	res, err := tx.ExecContext(ctx, `
		UPDATE dining_tables SET version = version + 1
		WHERE id = ? AND version = ?`, primary.ID, primary.Version)
	if err != nil {
		return fmt.Errorf("combinar mesas: %w", err)
	}
	if err := requireOneRow(res); err != nil {
		return err
	}

	for _, o := range others {
		res, err := tx.ExecContext(ctx, `
			UPDATE dining_tables SET merged_into_id = ?, version = version + 1
			WHERE id = ? AND version = ?`, primary.ID, o.ID, o.Version)
		if err != nil {
			return fmt.Errorf("combinar mesas: %w", err)
		}
		if err := requireOneRow(res); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("combinar mesas: %w", err)
	}
	return nil
}

func (r *TableRepository) Clean(ctx context.Context, ref port.TableRef) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("marcar mesa limpia: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE dining_tables
		SET status = 'FREE', current_guests = 0, assigned_waiter_id = NULL, version = version + 1
		WHERE id = ? AND version = ?`, ref.ID, ref.Version)
	if err != nil {
		return fmt.Errorf("marcar mesa limpia: %w", err)
	}
	if err := requireOneRow(res); err != nil {
		return err
	}

	// Las mesas que estaban combinadas a esta quedan libres otra vez.
	if _, err := tx.ExecContext(ctx, `
		UPDATE dining_tables SET merged_into_id = NULL, version = version + 1
		WHERE merged_into_id = ?`, ref.ID); err != nil {
		return fmt.Errorf("liberar mesas combinadas: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("marcar mesa limpia: %w", err)
	}
	return nil
}

func (r *TableRepository) Release(ctx context.Context, ref port.TableRef) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE dining_tables
		SET status = 'CLEANING', current_guests = 0, assigned_waiter_id = NULL, version = version + 1
		WHERE id = ? AND version = ? AND status = 'OCCUPIED'
		  AND NOT EXISTS (SELECT 1 FROM orders WHERE table_id = dining_tables.id AND status = 'OPEN')`,
		ref.ID, ref.Version)
	if err != nil {
		return fmt.Errorf("liberar mesa: %w", err)
	}
	return requireOneRow(res)
}

// requireOneRow convierte "0 filas afectadas" en ErrVersionConflict.
func requireOneRow(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("filas afectadas: %w", err)
	}
	if n == 0 {
		return port.ErrVersionConflict
	}
	return nil
}
