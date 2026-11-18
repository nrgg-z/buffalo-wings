package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/auth/port"
)

// fakeRepo imita la base de datos guardando un solo usuario en memoria.
type fakeRepo struct{ user *domain.User }

func (f *fakeRepo) FindByEmployeeCode(ctx context.Context, code string) (*domain.User, error) {
	if f.user == nil || f.user.EmployeeCode != code {
		return nil, port.ErrUserNotFound
	}
	copia := *f.user
	return &copia, nil
}

func (f *fakeRepo) UpdateLoginState(ctx context.Context, u *domain.User) error {
	f.user.FailedAttempts = u.FailedAttempts
	f.user.LockedUntil = u.LockedUntil
	return nil
}

func newTestService(t *testing.T) *AuthService {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakeRepo{user: &domain.User{
		ID:           1,
		EmployeeCode: "EMP-03",
		FullName:     "Carlos R.",
		Role:         domain.RoleWaiter,
		PinHash:      string(hash),
		Active:       true,
	}}

	svc, err := NewAuthService(repo, Config{
		PINLength:         4,
		MaxFailedAttempts: 3,
		LockoutDuration:   5 * time.Minute,
		TokenTTL:          time.Hour,
		JWTSecret:         []byte("0123456789abcdef0123456789abcdef"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestLoginExitoso(t *testing.T) {
	svc := newTestService(t)

	res, err := svc.Login(context.Background(), "EMP-03", "1234")
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if res.Token == "" {
		t.Error("el token no debería estar vacío")
	}
	if res.Role != domain.RoleWaiter {
		t.Errorf("rol esperado WAITER, llegó %s", res.Role)
	}
}

func TestUsuarioInexistenteDaCredencialesInvalidas(t *testing.T) {
	svc := newTestService(t)

	_, err := svc.Login(context.Background(), "EMP-99", "1234")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("esperaba ErrInvalidCredentials, llegó %v", err)
	}
}

func TestTresIntentosFallidosBloquean(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	for i := 1; i <= 2; i++ {
		_, err := svc.Login(ctx, "EMP-03", "0000")
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("intento %d: esperaba ErrInvalidCredentials, llegó %v", i, err)
		}
	}

	_, err := svc.Login(ctx, "EMP-03", "0000")
	if !errors.Is(err, domain.ErrUserLocked) {
		t.Fatalf("el tercer intento debía bloquear, llegó %v", err)
	}

	// Aunque ahora use el PIN correcto, sigue bloqueado.
	_, err = svc.Login(ctx, "EMP-03", "1234")
	if !errors.Is(err, domain.ErrUserLocked) {
		t.Errorf("debía seguir bloqueado, llegó %v", err)
	}
}

func TestBloqueoExpiraDespuesDeCincoMinutos(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		svc.Login(ctx, "EMP-03", "0000")
	}

	// Simulamos que pasaron 6 minutos.
	svc.now = func() time.Time { return time.Now().Add(6 * time.Minute) }

	if _, err := svc.Login(ctx, "EMP-03", "1234"); err != nil {
		t.Errorf("pasado el bloqueo debía entrar, llegó %v", err)
	}
}
