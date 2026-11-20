package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	auditdomain "buffalo-pos/internal/audit/domain"
	"buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/auth/port"
)

// maxAuditedCodeLen limita cuánto del employee_code escrito se guarda en la bitácora
// cuando el código no existe (es la longitud de users.employee_code).
const maxAuditedCodeLen = 20

// Razones de un login fallido, guardadas en audit_logs.detail.reason.
const (
	reasonInvalidFormat = "invalid_pin_format"
	reasonUnknownUser   = "unknown_user"
	reasonInactive      = "inactive"
	reasonLocked        = "locked"
	reasonBadPIN        = "bad_pin"
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

// AuthService contiene la lógica de inicio de sesión y de verificación de tokens.
type AuthService struct {
	users port.UserRepository
	audit port.AuditRecorder
	cfg   Config
	now   func() time.Time
}

func NewAuthService(users port.UserRepository, audit port.AuditRecorder, cfg Config) (*AuthService, error) {
	if users == nil || audit == nil {
		return nil, errors.New("el servicio de autenticación necesita repositorio de usuarios y bitácora")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, errors.New("el secreto JWT debe tener al menos 32 caracteres")
	}
	return &AuthService{users: users, audit: audit, cfg: cfg, now: time.Now}, nil
}

// Login valida el PIN de un empleado y devuelve un token si es correcto.
// Cada intento (exitoso o no) queda registrado en la bitácora con la IP del cliente.
//
// La auditoría es estricta: si no se puede escribir en audit_logs, el login
// falla con un error interno, para que nunca haya un acceso sin registro (RF-07).
func (s *AuthService) Login(ctx context.Context, employeeCode, pin, ip string) (*LoginResult, error) {
	if !validPIN(pin, s.cfg.PINLength) {
		if err := s.recordFailure(ctx, nil, employeeCode, ip, reasonInvalidFormat, nil); err != nil {
			return nil, err
		}
		return nil, domain.ErrInvalidCredentials
	}

	user, err := s.users.FindByEmployeeCode(ctx, employeeCode)
	if err != nil {
		if errors.Is(err, port.ErrUserNotFound) {
			if err := s.recordFailure(ctx, nil, employeeCode, ip, reasonUnknownUser, nil); err != nil {
				return nil, err
			}
			// Mismo error que un PIN malo, para no revelar qué códigos existen.
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	now := s.now()

	if !user.Active {
		if err := s.recordFailure(ctx, user, user.EmployeeCode, ip, reasonInactive, nil); err != nil {
			return nil, err
		}
		return nil, domain.ErrUserInactive
	}
	if user.IsLocked(now) {
		if err := s.recordFailure(ctx, user, user.EmployeeCode, ip, reasonLocked, nil); err != nil {
			return nil, err
		}
		return nil, domain.ErrUserLocked
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PinHash), []byte(pin)) != nil {
		return nil, s.handleBadPIN(ctx, user, ip, now)
	}

	user.RegisterSuccessfulLogin()
	if err := s.users.UpdateLoginState(ctx, user); err != nil {
		return nil, err
	}

	token, expires, err := s.issueToken(user, now)
	if err != nil {
		return nil, err
	}

	if err := s.record(ctx, auditdomain.ActionLoginSuccess, &user.ID, ip, map[string]any{
		"employee_code": user.EmployeeCode,
	}); err != nil {
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

// handleBadPIN suma el intento fallido, lo registra y, si se llegó al máximo,
// registra también el bloqueo. Devuelve el error que debe ver el cliente.
func (s *AuthService) handleBadPIN(ctx context.Context, user *domain.User, ip string, now time.Time) error {
	user.RegisterFailedAttempt(now, s.cfg.MaxFailedAttempts, s.cfg.LockoutDuration)
	if err := s.users.UpdateLoginState(ctx, user); err != nil {
		return err
	}

	if err := s.recordFailure(ctx, user, user.EmployeeCode, ip, reasonBadPIN, map[string]any{
		"failed_attempts": user.FailedAttempts,
	}); err != nil {
		return err
	}

	if !user.IsLocked(now) {
		return domain.ErrInvalidCredentials
	}

	if err := s.record(ctx, auditdomain.ActionUserLocked, &user.ID, ip, map[string]any{
		"employee_code":   user.EmployeeCode,
		"failed_attempts": user.FailedAttempts,
		"locked_until":    user.LockedUntil.UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}
	return domain.ErrUserLocked
}

// VerifyToken revisa firma, algoritmo y expiración de un token y devuelve
// quién es su dueño. Cualquier problema se reporta como domain.ErrInvalidToken,
// sin dar detalles que le sirvan a un atacante.
//
// No consulta la base de datos: así cada petición autenticada es rápida (RNF-01).
func (s *AuthService) VerifyToken(tokenString string) (domain.Principal, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (any, error) { return s.cfg.JWTSecret, nil },
		// Solo HS256: evita el ataque de cambiar el algoritmo ("alg": "none" u otro).
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		// Un token sin fecha de expiración se rechaza.
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(s.now),
	)
	if err != nil {
		return domain.Principal{}, fmt.Errorf("%w: %v", domain.ErrInvalidToken, err)
	}

	if claims.UserID <= 0 || !claims.Role.IsValid() {
		return domain.Principal{}, fmt.Errorf("%w: claims incompletos", domain.ErrInvalidToken)
	}

	return domain.Principal{
		UserID:       claims.UserID,
		EmployeeCode: claims.EmployeeCode,
		Role:         claims.Role,
		ExpiresAt:    claims.ExpiresAt.Time,
	}, nil
}

// CurrentUser devuelve los datos actuales del empleado dueño del token.
// A diferencia de VerifyToken, sí consulta la base de datos, para que /auth/me
// muestre el nombre y estado reales aunque hayan cambiado después del login.
func (s *AuthService) CurrentUser(ctx context.Context, userID int64) (*domain.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, port.ErrUserNotFound) {
			// El token era válido pero el usuario ya no existe.
			return nil, domain.ErrInvalidToken
		}
		return nil, err
	}
	if !user.Active {
		return nil, domain.ErrUserInactive
	}
	return user, nil
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

// recordFailure registra un LOGIN_FAILED. user puede ser nil cuando no se sabe
// quién intentó entrar; en ese caso se guarda el código escrito (recortado).
// El PIN nunca se guarda.
func (s *AuthService) recordFailure(ctx context.Context, user *domain.User, employeeCode, ip, reason string, extra map[string]any) error {
	detail := map[string]any{
		"employee_code": truncate(employeeCode, maxAuditedCodeLen),
		"reason":        reason,
	}
	for k, v := range extra {
		detail[k] = v
	}

	var userID *int64
	if user != nil {
		userID = &user.ID
	}
	return s.record(ctx, auditdomain.ActionLoginFailed, userID, ip, detail)
}

// record escribe una entrada sobre un usuario en la bitácora.
func (s *AuthService) record(ctx context.Context, action string, userID *int64, ip string, detail map[string]any) error {
	err := s.audit.Record(ctx, auditdomain.Entry{
		UserID:     userID,
		Action:     action,
		EntityType: auditdomain.EntityUser,
		EntityID:   userID,
		Detail:     detail,
		IPAddress:  ip,
	})
	if err != nil {
		return fmt.Errorf("registrar %s en bitácora: %w", action, err)
	}
	return nil
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

// truncate recorta s a un máximo de n caracteres (runas, no bytes, para no
// partir una letra con tilde a la mitad).
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
