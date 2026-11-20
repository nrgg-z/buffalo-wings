package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	auditdomain "buffalo-pos/internal/audit/domain"
	"buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/auth/port"
)

const (
	testSecret = "0123456789abcdef0123456789abcdef"
	testIP     = "192.168.1.50"
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

func (f *fakeRepo) FindByID(ctx context.Context, id int64) (*domain.User, error) {
	if f.user == nil || f.user.ID != id {
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

// fakeAudit imita la bitácora guardando las entradas en una lista.
// Si err no es nil, simula que la base de datos falló.
type fakeAudit struct {
	entries []auditdomain.Entry
	err     error
}

func (f *fakeAudit) Record(ctx context.Context, e auditdomain.Entry) error {
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, e)
	return nil
}

// actions devuelve solo los nombres de acción registrados, en orden.
func (f *fakeAudit) actions() []string {
	out := make([]string, len(f.entries))
	for i, e := range f.entries {
		out[i] = e.Action
	}
	return out
}

type testEnv struct {
	svc   *AuthService
	repo  *fakeRepo
	audit *fakeAudit
}

func newTestEnv(t *testing.T) *testEnv {
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
	audit := &fakeAudit{}

	svc, err := NewAuthService(repo, audit, Config{
		PINLength:         4,
		MaxFailedAttempts: 3,
		LockoutDuration:   5 * time.Minute,
		TokenTTL:          time.Hour,
		JWTSecret:         []byte(testSecret),
	})
	if err != nil {
		t.Fatal(err)
	}
	return &testEnv{svc: svc, repo: repo, audit: audit}
}

// ---------- Login (pruebas originales, adaptadas a la nueva firma) ----------

func TestLoginExitoso(t *testing.T) {
	env := newTestEnv(t)

	res, err := env.svc.Login(context.Background(), "EMP-03", "1234", testIP)
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
	env := newTestEnv(t)

	_, err := env.svc.Login(context.Background(), "EMP-99", "1234", testIP)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("esperaba ErrInvalidCredentials, llegó %v", err)
	}
}

func TestTresIntentosFallidosBloquean(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	for i := 1; i <= 2; i++ {
		_, err := env.svc.Login(ctx, "EMP-03", "0000", testIP)
		if !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("intento %d: esperaba ErrInvalidCredentials, llegó %v", i, err)
		}
	}

	_, err := env.svc.Login(ctx, "EMP-03", "0000", testIP)
	if !errors.Is(err, domain.ErrUserLocked) {
		t.Fatalf("el tercer intento debía bloquear, llegó %v", err)
	}

	// Aunque ahora use el PIN correcto, sigue bloqueado.
	_, err = env.svc.Login(ctx, "EMP-03", "1234", testIP)
	if !errors.Is(err, domain.ErrUserLocked) {
		t.Errorf("debía seguir bloqueado, llegó %v", err)
	}
}

func TestBloqueoExpiraDespuesDeCincoMinutos(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		env.svc.Login(ctx, "EMP-03", "0000", testIP)
	}

	// Simulamos que pasaron 6 minutos.
	env.svc.now = func() time.Time { return time.Now().Add(6 * time.Minute) }

	if _, err := env.svc.Login(ctx, "EMP-03", "1234", testIP); err != nil {
		t.Errorf("pasado el bloqueo debía entrar, llegó %v", err)
	}
}

func TestUsuarioInactivoNoEntra(t *testing.T) {
	env := newTestEnv(t)
	env.repo.user.Active = false

	_, err := env.svc.Login(context.Background(), "EMP-03", "1234", testIP)
	if !errors.Is(err, domain.ErrUserInactive) {
		t.Errorf("esperaba ErrUserInactive, llegó %v", err)
	}
}

// ---------- Bitácora del login ----------

func TestLoginExitosoQuedaEnBitacora(t *testing.T) {
	env := newTestEnv(t)

	if _, err := env.svc.Login(context.Background(), "EMP-03", "1234", testIP); err != nil {
		t.Fatal(err)
	}

	if len(env.audit.entries) != 1 {
		t.Fatalf("esperaba 1 entrada, llegaron %d", len(env.audit.entries))
	}
	e := env.audit.entries[0]
	if e.Action != auditdomain.ActionLoginSuccess {
		t.Errorf("acción esperada LOGIN_SUCCESS, llegó %s", e.Action)
	}
	if e.UserID == nil || *e.UserID != 1 {
		t.Errorf("user_id esperado 1, llegó %v", e.UserID)
	}
	if e.IPAddress != testIP {
		t.Errorf("IP esperada %s, llegó %s", testIP, e.IPAddress)
	}
	if e.EntityType != auditdomain.EntityUser {
		t.Errorf("entity_type esperado user, llegó %s", e.EntityType)
	}
}

func TestPINIncorrectoQuedaEnBitacoraConIntentos(t *testing.T) {
	env := newTestEnv(t)

	env.svc.Login(context.Background(), "EMP-03", "0000", testIP)

	if len(env.audit.entries) != 1 {
		t.Fatalf("esperaba 1 entrada, llegaron %d", len(env.audit.entries))
	}
	e := env.audit.entries[0]
	if e.Action != auditdomain.ActionLoginFailed {
		t.Errorf("acción esperada LOGIN_FAILED, llegó %s", e.Action)
	}
	if e.Detail["reason"] != reasonBadPIN {
		t.Errorf("razón esperada bad_pin, llegó %v", e.Detail["reason"])
	}
	if e.Detail["failed_attempts"] != 1 {
		t.Errorf("failed_attempts esperado 1, llegó %v", e.Detail["failed_attempts"])
	}
}

func TestTercerIntentoRegistraFalloYBloqueo(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		env.svc.Login(ctx, "EMP-03", "0000", testIP)
	}

	got := strings.Join(env.audit.actions(), ",")
	want := "LOGIN_FAILED,LOGIN_FAILED,LOGIN_FAILED,USER_LOCKED"
	if got != want {
		t.Errorf("acciones esperadas %s, llegaron %s", want, got)
	}
	if env.audit.entries[3].Detail["locked_until"] == nil {
		t.Error("USER_LOCKED debía incluir locked_until")
	}
}

func TestIntentoEstandoBloqueadoQuedaEnBitacora(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		env.svc.Login(ctx, "EMP-03", "0000", testIP)
	}
	env.svc.Login(ctx, "EMP-03", "1234", testIP)

	last := env.audit.entries[len(env.audit.entries)-1]
	if last.Action != auditdomain.ActionLoginFailed || last.Detail["reason"] != reasonLocked {
		t.Errorf("esperaba LOGIN_FAILED con razón locked, llegó %s %v", last.Action, last.Detail["reason"])
	}
}

func TestUsuarioInexistenteSeRegistraSinUserID(t *testing.T) {
	env := newTestEnv(t)

	env.svc.Login(context.Background(), "EMP-99", "1234", testIP)

	if len(env.audit.entries) != 1 {
		t.Fatalf("esperaba 1 entrada, llegaron %d", len(env.audit.entries))
	}
	e := env.audit.entries[0]
	if e.UserID != nil {
		t.Errorf("user_id debía ser nil, llegó %v", *e.UserID)
	}
	if e.Detail["employee_code"] != "EMP-99" || e.Detail["reason"] != reasonUnknownUser {
		t.Errorf("detalle inesperado: %v", e.Detail)
	}
}

func TestCodigoLargoSeRecortaEnBitacora(t *testing.T) {
	env := newTestEnv(t)

	env.svc.Login(context.Background(), strings.Repeat("X", 200), "1234", testIP)

	code, _ := env.audit.entries[0].Detail["employee_code"].(string)
	if len([]rune(code)) != maxAuditedCodeLen {
		t.Errorf("el código debía recortarse a %d, quedó de %d", maxAuditedCodeLen, len(code))
	}
}

func TestLaBitacoraNuncaGuardaElPIN(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	env.svc.Login(ctx, "EMP-03", "9876", testIP) // PIN malo
	env.svc.Login(ctx, "EMP-03", "1234", testIP) // PIN bueno
	env.svc.Login(ctx, "EMP-99", "5555", testIP) // usuario inexistente
	env.svc.Login(ctx, "EMP-03", "12a", testIP)  // formato inválido

	for _, e := range env.audit.entries {
		texto := fmt.Sprintf("%v", e.Detail)
		for _, pin := range []string{"9876", "1234", "5555", "12a"} {
			if strings.Contains(texto, pin) {
				t.Errorf("la entrada %s contiene el PIN %q: %s", e.Action, pin, texto)
			}
		}
	}
}

func TestSiFallaLaBitacoraElLoginFalla(t *testing.T) {
	env := newTestEnv(t)
	dbErr := errors.New("mysql caído")
	env.audit.err = dbErr

	res, err := env.svc.Login(context.Background(), "EMP-03", "1234", testIP)
	if res != nil {
		t.Error("no debía devolver token si no se pudo auditar")
	}
	if !errors.Is(err, dbErr) {
		t.Errorf("esperaba el error de la bitácora envuelto, llegó %v", err)
	}
	// Debe ser un error interno, no uno de negocio.
	if errors.Is(err, domain.ErrInvalidCredentials) {
		t.Error("un fallo de bitácora no debía parecer credenciales inválidas")
	}
}

// ---------- VerifyToken ----------

func TestVerifyTokenValidoDevuelvePrincipal(t *testing.T) {
	env := newTestEnv(t)
	res, err := env.svc.Login(context.Background(), "EMP-03", "1234", testIP)
	if err != nil {
		t.Fatal(err)
	}

	p, err := env.svc.VerifyToken(res.Token)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if p.UserID != 1 || p.EmployeeCode != "EMP-03" || p.Role != domain.RoleWaiter {
		t.Errorf("principal inesperado: %+v", p)
	}
	if !p.ExpiresAt.Equal(res.ExpiresAt.Truncate(time.Second)) {
		t.Errorf("ExpiresAt esperado %v, llegó %v", res.ExpiresAt, p.ExpiresAt)
	}
}

func TestVerifyTokenExpiradoSeRechaza(t *testing.T) {
	env := newTestEnv(t)
	res, _ := env.svc.Login(context.Background(), "EMP-03", "1234", testIP)

	env.svc.now = func() time.Time { return time.Now().Add(61 * time.Minute) }

	if _, err := env.svc.VerifyToken(res.Token); !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("esperaba ErrInvalidToken, llegó %v", err)
	}
}

// signToken firma claims arbitrarios para simular tokens manipulados.
func signToken(t *testing.T, method jwt.SigningMethod, key any, claims jwt.Claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func validClaims() Claims {
	now := time.Now()
	return Claims{
		UserID:       1,
		EmployeeCode: "EMP-03",
		Role:         domain.RoleWaiter,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
}

func TestVerifyTokenRechazaTokensManipulados(t *testing.T) {
	env := newTestEnv(t)

	sinExp := validClaims()
	sinExp.ExpiresAt = nil

	rolFalso := validClaims()
	rolFalso.Role = "SUPERUSER"

	sinUsuario := validClaims()
	sinUsuario.UserID = 0

	casos := map[string]string{
		"firmado con otra clave": signToken(t, jwt.SigningMethodHS256, []byte("otra-clave-otra-clave-otra-clave!"), validClaims()),
		"algoritmo HS384":        signToken(t, jwt.SigningMethodHS384, []byte(testSecret), validClaims()),
		"algoritmo none":         signToken(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"sin expiración":         signToken(t, jwt.SigningMethodHS256, []byte(testSecret), sinExp),
		"rol inexistente":        signToken(t, jwt.SigningMethodHS256, []byte(testSecret), rolFalso),
		"sin uid":                signToken(t, jwt.SigningMethodHS256, []byte(testSecret), sinUsuario),
		"basura":                 "esto.no.es-un-jwt",
		"vacío":                  "",
	}

	for nombre, token := range casos {
		t.Run(nombre, func(t *testing.T) {
			if _, err := env.svc.VerifyToken(token); !errors.Is(err, domain.ErrInvalidToken) {
				t.Errorf("esperaba ErrInvalidToken, llegó %v", err)
			}
		})
	}
}

// ---------- CurrentUser ----------

func TestCurrentUserDevuelveDatosFrescos(t *testing.T) {
	env := newTestEnv(t)

	u, err := env.svc.CurrentUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if u.FullName != "Carlos R." {
		t.Errorf("nombre esperado Carlos R., llegó %s", u.FullName)
	}
}

func TestCurrentUserInexistenteEsTokenInvalido(t *testing.T) {
	env := newTestEnv(t)

	if _, err := env.svc.CurrentUser(context.Background(), 99); !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("esperaba ErrInvalidToken, llegó %v", err)
	}
}

func TestCurrentUserInactivo(t *testing.T) {
	env := newTestEnv(t)
	env.repo.user.Active = false

	if _, err := env.svc.CurrentUser(context.Background(), 1); !errors.Is(err, domain.ErrUserInactive) {
		t.Errorf("esperaba ErrUserInactive, llegó %v", err)
	}
}

// ---------- Constructor ----------

func TestNewAuthServiceRechazaSecretoCorto(t *testing.T) {
	_, err := NewAuthService(&fakeRepo{}, &fakeAudit{}, Config{JWTSecret: []byte("corto")})
	if err == nil {
		t.Error("debía rechazar un secreto de menos de 32 caracteres")
	}
}
