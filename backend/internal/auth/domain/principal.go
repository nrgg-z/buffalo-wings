package domain

import "time"

// Principal es "quién está haciendo la petición": se obtiene del token JWT
// y viaja en el context de cada request autenticada.
// Es el equivalente a req.user en Express con Passport.
type Principal struct {
	UserID       int64
	EmployeeCode string
	Role         Role
	ExpiresAt    time.Time // cuándo vence el token con el que se autenticó
}

// HasAnyRole indica si el principal tiene alguno de los roles permitidos.
//
// Regla del negocio: ADMIN puede hacer todo, así que siempre pasa sin necesidad
// de listarlo en cada ruta. MANAGER, en cambio, se debe listar explícitamente
// en las rutas que le correspondan.
func (p Principal) HasAnyRole(allowed ...Role) bool {
	if p.Role == RoleAdmin {
		return true
	}
	for _, r := range allowed {
		if p.Role == r {
			return true
		}
	}
	return false
}
