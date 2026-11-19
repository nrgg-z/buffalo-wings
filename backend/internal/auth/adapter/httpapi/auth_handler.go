package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/auth/service"
	"buffalo-pos/internal/platform/httpx"
)

// maxLoginBodyBytes limita el tamaño del cuerpo del login (un JSON con código y PIN).
const maxLoginBodyBytes = 1 << 10

// AuthService es lo que el handler necesita del servicio de autenticación.
type AuthService interface {
	TokenVerifier
	Login(ctx context.Context, employeeCode, pin, ip string) (*service.LoginResult, error)
	CurrentUser(ctx context.Context, userID int64) (*domain.User, error)
}

// AuthHandler expone las rutas HTTP de autenticación.
type AuthHandler struct {
	auth AuthService
}

func NewAuthHandler(auth AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register agrega las rutas de autenticación al router.
// loginGuard envuelve el login (ej. el límite de intentos por IP); puede ser nil.
func (h *AuthHandler) Register(mux *http.ServeMux, loginGuard func(http.Handler) http.Handler) {
	var login http.Handler = http.HandlerFunc(h.login)
	if loginGuard != nil {
		login = loginGuard(login)
	}

	mux.Handle("POST /api/v1/auth/login", login)
	mux.Handle("GET /api/v1/auth/me", Protected(h.auth)(http.HandlerFunc(h.me)))
}

type loginRequest struct {
	EmployeeCode string `json:"employee_code"`
	PIN          string `json:"pin"`
}

type userResponse struct {
	ID           int64  `json:"id"`
	EmployeeCode string `json:"employee_code"`
	FullName     string `json:"full_name"`
	Role         string `json:"role"`
}

type loginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
	User      userResponse `json:"user"`
}

type meResponse struct {
	User           userResponse `json:"user"`
	TokenExpiresAt time.Time    `json:"token_expires_at"`
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxLoginBodyBytes)

	var req loginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "El cuerpo de la petición no es válido")
		return
	}
	if req.EmployeeCode == "" || req.PIN == "" {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Se requiere employee_code y pin")
		return
	}

	res, err := h.auth.Login(r.Context(), req.EmployeeCode, req.PIN, httpx.ClientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Código de empleado o PIN incorrecto")
		case errors.Is(err, domain.ErrUserLocked):
			httpx.WriteError(w, http.StatusLocked, "user_locked", "Usuario bloqueado temporalmente. Intenta más tarde")
		case errors.Is(err, domain.ErrUserInactive):
			httpx.WriteError(w, http.StatusForbidden, "user_inactive", "El usuario está inactivo")
		default:
			log.Printf("error en login: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Error interno del servidor")
		}
		return
	}

	httpx.WriteJSON(w, http.StatusOK, loginResponse{
		Token:     res.Token,
		ExpiresAt: res.ExpiresAt.UTC(),
		User: userResponse{
			ID:           res.UserID,
			EmployeeCode: res.EmployeeCode,
			FullName:     res.FullName,
			Role:         string(res.Role),
		},
	})
}

// me devuelve los datos del empleado dueño del token.
func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	p, ok := PrincipalFrom(r.Context())
	if !ok {
		// No debería pasar: la ruta está protegida con RequireAuth.
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized", "Se requiere iniciar sesión")
		return
	}

	user, err := h.auth.CurrentUser(r.Context(), p.UserID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidToken):
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_token", "La sesión no es válida o expiró. Inicia sesión de nuevo")
		case errors.Is(err, domain.ErrUserInactive):
			httpx.WriteError(w, http.StatusForbidden, "user_inactive", "El usuario está inactivo")
		default:
			log.Printf("error en /auth/me: %v", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal_error", "Error interno del servidor")
		}
		return
	}

	httpx.WriteJSON(w, http.StatusOK, meResponse{
		User: userResponse{
			ID:           user.ID,
			EmployeeCode: user.EmployeeCode,
			FullName:     user.FullName,
			Role:         string(user.Role),
		},
		TokenExpiresAt: p.ExpiresAt.UTC(),
	})
}
