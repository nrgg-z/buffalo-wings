package httpx

import (
	"net/http"
	"strconv"
)

// corsMaxAge es cuánto puede el navegador recordar la respuesta del preflight.
const corsMaxAge = 600 // segundos

// CORS permite que el frontend (otro origen, ej. http://localhost:5173) llame a la API.
//
// Solo se responde con Access-Control-Allow-Origin cuando el Origin está en la
// lista de app.yaml; a cualquier otro origen el navegador le bloquea la respuesta.
// Las peticiones OPTIONS de preflight (las que el navegador manda antes de un
// POST con JSON o de un GET con Authorization) se contestan aquí con 204.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			_, ok := allowed[origin]

			// La respuesta cambia según el Origin; esto evita que un caché la mezcle.
			w.Header().Add("Vary", "Origin")

			if ok {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				h.Set("Access-Control-Max-Age", strconv.Itoa(corsMaxAge))
			}

			// Preflight: no llega a los handlers de negocio.
			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				if ok {
					w.WriteHeader(http.StatusNoContent)
				} else {
					w.WriteHeader(http.StatusForbidden)
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
