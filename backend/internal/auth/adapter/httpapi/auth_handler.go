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
)

// Authenticator es lo que el handler necesita del servicio de login.
type Authenticator interface {
	Login(ctx context.Context, employeeCode, pin string) (*service.LoginResult, error)
}

// AuthHandler expone las rutas HTTP de autenticación.
type AuthHandler struct {
	auth Authenticator
}

func NewAuthHandler(auth Authenticator) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// Register agrega las rutas de autenticación al router.
func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
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

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10)

	var req loginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "El cuerpo de la petición no es válido")
		return
	}
	if req.EmployeeCode == "" || req.PIN == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "Se requiere employee_code y pin")
		return
	}

	res, err := h.auth.Login(r.Context(), req.EmployeeCode, req.PIN)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "Código de empleado o PIN incorrecto")
		case errors.Is(err, domain.ErrUserLocked):
			writeError(w, http.StatusLocked, "user_locked", "Usuario bloqueado temporalmente. Intenta más tarde")
		case errors.Is(err, domain.ErrUserInactive):
			writeError(w, http.StatusForbidden, "user_inactive", "El usuario está inactivo")
		default:
			log.Printf("error en login: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Error interno del servidor")
		}
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token:     res.Token,
		ExpiresAt: res.ExpiresAt,
		User: userResponse{
			ID:           res.UserID,
			EmployeeCode: res.EmployeeCode,
			FullName:     res.FullName,
			Role:         string(res.Role),
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("error escribiendo respuesta: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: code, Message: message})
}
