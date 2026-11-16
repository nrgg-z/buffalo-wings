package port

import (
	"context"
	"errors"

	"buffalo-pos/internal/auth/domain"
)

// ErrUserNotFound se devuelve cuando no existe un empleado con ese código.
var ErrUserNotFound = errors.New("usuario no encontrado")

// UserRepository es el contrato que necesita el servicio de autenticación.
type UserRepository interface {
	// FindByEmployeeCode busca un empleado (con su rol) por código, ej. "EMP-03".
	FindByEmployeeCode(ctx context.Context, code string) (*domain.User, error)

	// UpdateLoginState guarda los intentos fallidos y el bloqueo del usuario.
	UpdateLoginState(ctx context.Context, u *domain.User) error
}
