package domain

import (
	"errors"
	"time"
)

// Role representa el rol de un empleado en el sistema.
type Role string

const (
	RoleAdmin   Role = "ADMIN"
	RoleManager Role = "MANAGER"
	RoleHostess Role = "HOSTESS"
	RoleWaiter  Role = "WAITER"
	RoleKitchen Role = "KITCHEN"
	RoleCashier Role = "CASHIER"
)

// Errores de negocio de la autenticación.
var (
	ErrInvalidCredentials = errors.New("credenciales inválidas")
	ErrUserLocked         = errors.New("usuario bloqueado temporalmente")
	ErrUserInactive       = errors.New("usuario inactivo")
)

// User es el empleado que inicia sesión.
type User struct {
	ID             int64
	EmployeeCode   string
	FullName       string
	Role           Role
	PinHash        string
	Active         bool
	FailedAttempts int
	LockedUntil    *time.Time
}

// IsLocked indica si el usuario sigue bloqueado en este momento.
func (u *User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// RegisterFailedAttempt suma un intento fallido y bloquea al llegar al máximo.
func (u *User) RegisterFailedAttempt(now time.Time, maxAttempts int, lockout time.Duration) {
	// Si el bloqueo anterior ya venció, se empieza de cero.
	if u.LockedUntil != nil && !now.Before(*u.LockedUntil) {
		u.FailedAttempts = 0
		u.LockedUntil = nil
	}

	u.FailedAttempts++

	if u.FailedAttempts >= maxAttempts {
		until := now.Add(lockout)
		u.LockedUntil = &until
	}
}

// RegisterSuccessfulLogin limpia los intentos fallidos y el bloqueo.
func (u *User) RegisterSuccessfulLogin() {
	u.FailedAttempts = 0
	u.LockedUntil = nil
}
