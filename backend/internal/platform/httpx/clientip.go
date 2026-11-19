package httpx

import (
	"net"
	"net/http"
)

// ClientIP devuelve la IP de quien hizo la petición, tomada de la conexión TCP.
//
// A propósito NO se lee X-Forwarded-For: ese header lo puede escribir cualquier
// cliente, y si lo usáramos alguien podría saltarse el límite de intentos de
// login o ensuciar la bitácora con IPs falsas. Si en el futuro el backend queda
// detrás de un proxy (nginx), habrá que confiar solo en el header que ponga ese proxy.
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr sin puerto (raro, pero posible en pruebas): se usa tal cual.
		return r.RemoteAddr
	}
	return host
}
