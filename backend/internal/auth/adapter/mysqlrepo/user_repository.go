package mysqlrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/auth/port"
)

// UserRepository implementa port.UserRepository usando MySQL.
type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Verificación en compilación: si falta un método del contrato, no compila.
var _ port.UserRepository = (*UserRepository)(nil)

// selectUserQuery es la parte común de las búsquedas de usuario (con su rol).
const selectUserQuery = `
	SELECT u.id, u.employee_code, u.full_name, u.pin_hash,
	       r.code, u.is_active, u.failed_attempts, u.locked_until
	FROM users u
	JOIN roles r ON r.id = u.role_id`

const (
	findByEmployeeCodeQuery = selectUserQuery + ` WHERE u.employee_code = ?`
	findByIDQuery           = selectUserQuery + ` WHERE u.id = ?`
)

func (r *UserRepository) FindByEmployeeCode(ctx context.Context, code string) (*domain.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, findByEmployeeCodeQuery, code))
	if err != nil {
		if errors.Is(err, port.ErrUserNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("buscar usuario por código: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	u, err := scanUser(r.db.QueryRowContext(ctx, findByIDQuery, id))
	if err != nil {
		if errors.Is(err, port.ErrUserNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("buscar usuario por id: %w", err)
	}
	return u, nil
}

// scanUser convierte una fila de selectUserQuery en un domain.User.
func scanUser(row *sql.Row) (*domain.User, error) {
	var (
		u           domain.User
		role        string
		lockedUntil sql.NullTime
	)

	err := row.Scan(
		&u.ID,
		&u.EmployeeCode,
		&u.FullName,
		&u.PinHash,
		&role,
		&u.Active,
		&u.FailedAttempts,
		&lockedUntil,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, port.ErrUserNotFound
		}
		return nil, err
	}

	u.Role = domain.Role(role)
	if lockedUntil.Valid {
		t := lockedUntil.Time
		u.LockedUntil = &t
	}

	return &u, nil
}

func (r *UserRepository) UpdateLoginState(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET failed_attempts = ?, locked_until = ? WHERE id = ?`,
		u.FailedAttempts, u.LockedUntil, u.ID,
	)
	if err != nil {
		return fmt.Errorf("actualizar estado de login: %w", err)
	}
	return nil
}
