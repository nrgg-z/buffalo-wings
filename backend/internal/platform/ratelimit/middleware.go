package ratelimit

import (
	"log"
	"math"
	"net/http"
	"strconv"

	"buffalo-pos/internal/platform/httpx"
)

// Middleware limita las peticiones por IP del cliente. Si se pasa del límite,
// responde 429 con el header Retry-After (en segundos) y no llama al handler.
func Middleware(l *Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := httpx.ClientIP(r)

			ok, retryAfter := l.Allow(ip)
			if !ok {
				seconds := int(math.Ceil(retryAfter.Seconds()))
				w.Header().Set("Retry-After", strconv.Itoa(seconds))
				log.Printf("rate limit excedido: ip=%s ruta=%s", ip, r.URL.Path)
				httpx.WriteError(w, http.StatusTooManyRequests, "too_many_requests",
					"Demasiados intentos. Espera un momento e intenta de nuevo")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
