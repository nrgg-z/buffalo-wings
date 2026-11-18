package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/auth/port"
)

// Config reúne los valores que vienen de security.yaml y del entorno.
type Config struct {
	PINLength         int
	MaxFailedAttempts int
	LockoutDuration   time.Duration
	TokenTTL          time.Duration
	JWTSecret         []byte
}

// Claims es la información que viaja dentro del token JWT.
type Claims struct {
	UserID       int64       `json:"uid"`
	EmployeeCode string      `json:"emp"`
	Role         domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// LoginResult es lo que devuelve un login exitoso.
type LoginResult struct {
	Token        string
	ExpiresAt    time.Time
	UserID       int64
	EmployeeCode string
	FullName     string
	Role         domain.Role
}

// AuthService contiene la lógica de inicio de sesión.
type AuthService struct {
	users port.UserRepository
	cfg   Config
	now   func() time.Time
}

func NewAuthService(users port.UserRepository, cfg Config) (*AuthService, error) {
	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("el secreto JWT debe tener al menos 32 caracteres")
	}
	return &AuthService{users: users, cfg: cfg, now: time.Now}, nil
}

// Login valida el PIN de un empleado y devuelve un token si es correcto.
func (s *AuthService) Login(ctx context.Context, employeeCode, pin string) (*LoginResult, error) {
	if !validPIN(pin, s.cfg.PINLength) {
		return nil, domain.ErrInvalidCredentials
	}

	user, err := s.users.FindByEmployeeCode(ctx, employeeCode)
	if err != nil {
		if errors.Is(err, port.ErrUserNotFound) {
			// Mismo error que un PIN malo, para no revelar qué códigos existen.
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	now := s.now()

	if !user.Active {
		return nil, domain.ErrUserInactive
	}
	if user.IsLocked(now) {
		return nil, domain.ErrUserLocked
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PinHash), []byte(pin)) != nil {
		user.RegisterFailedAttempt(now, s.cfg.MaxFailedAttempts, s.cfg.LockoutDuration)
		if updErr := s.users.UpdateLoginState(ctx, user); updErr != nil {
			return nil, updErr
		}
		if user.IsLocked(now) {
			return nil, domain.ErrUserLocked
		}
		return nil, domain.ErrInvalidCredentials
	}

	user.RegisterSuccessfulLogin()
	if err := s.users.UpdateLoginState(ctx, user); err != nil {
		return nil, err
	}

	token, expires, err := s.issueToken(user, now)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:        token,
		ExpiresAt:    expires,
		UserID:       user.ID,
		EmployeeCode: user.EmployeeCode,
		FullName:     user.FullName,
		Role:         user.Role,
	}, nil
}

func (s *AuthService) issueToken(u *domain.User, now time.Time) (string, time.Time, error) {
	expires := now.Add(s.cfg.TokenTTL)

	claims := Claims{
		UserID:       u.ID,
		EmployeeCode: u.EmployeeCode,
		Role:         u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.EmployeeCode,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.cfg.JWTSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("firmar token: %w", err)
	}
	return signed, expires, nil
}

// validPIN revisa que el PIN tenga la longitud pedida y solo dígitos.
func validPIN(pin string, length int) bool {
	if len(pin) != length {
		return false
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
