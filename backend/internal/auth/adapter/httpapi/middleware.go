package httpapi

import (
	"context"
	"net/http"
	"strings"

	"buffalo-pos/internal/auth/domain"
	"buffalo-pos/internal/platform/httpx"
)

// TokenVerifier es lo que el middleware necesita del servicio: validar un token.
type TokenVerifier interface {
	VerifyToken(token string) (domain.Principal, error)
}

// ctxKey es privado para que ningún otro paquete pueda pisar el valor guardado.
type ctxKey struct{}

// PrincipalFrom devuelve quién hizo la petición. Solo funciona en handlers
// protegidos con RequireAuth; si no, ok es false.
//
// Uso desde otro módulo:  p, ok := authhttp.PrincipalFrom(r.Context())
func PrincipalFrom(ctx context.Context) (domain.Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(domain.Principal)
	return p, ok
}

// withPrincipal guarda el principal en el context de la petición.
func withPrincipal(ctx context.Context, p domain.Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// RequireAuth exige un header "Authorization: Bearer <token>" válido.
// Si el token falta, está mal formado, fue manipulado o expiró, responde 401.
// Si es válido, deja el Principal en el context y llama al siguiente handler.
func RequireAuth(v TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized",
					"Se requiere iniciar sesión")
				return
			}

			p, err := v.VerifyToken(token)
			if err != nil {
				// El detalle (firma, expiración...) no se envía al cliente.
				httpx.WriteError(w, http.StatusUnauthorized, "invalid_token",
					"La sesión no es válida o expiró. Inicia sesión de nuevo")
				return
			}

			next.ServeHTTP(w, r.WithContext(withPrincipal(r.Context(), p)))
		})
	}
}

// RequireRoles deja pasar solo a los roles indicados (ADMIN siempre pasa).
// Debe ir DESPUÉS de RequireAuth; si no hay principal, responde 401.
func RequireRoles(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized",
					"Se requiere iniciar sesión")
				return
			}
			if !p.HasAnyRole(roles...) {
				httpx.WriteError(w, http.StatusForbidden, "forbidden",
					"No tienes permiso para realizar esta acción")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Protected combina RequireAuth y RequireRoles en una sola llamada, para que
// registrar rutas en los demás módulos sea corto y legible:
//
//	mux.Handle("POST /api/v1/tables/{id}/assign",
//	    authhttp.Protected(verifier, domain.RoleHostess, domain.RoleManager)(handler))
//
// Si no se pasan roles, basta con estar autenticado.
func Protected(v TokenVerifier, roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if len(roles) > 0 {
			next = RequireRoles(roles...)(next)
		}
		return RequireAuth(v)(next)
	}
}

// bearerToken extrae el token del header Authorization.
// El esquema "Bearer" se acepta sin importar mayúsculas (lo dice el RFC 6750).
func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}
