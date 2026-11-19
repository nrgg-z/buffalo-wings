// Package domain define qué es una entrada de la bitácora de auditoría (RF-07).
package domain

import (
	"errors"
	"fmt"
)

// Acciones registradas en la bitácora. Se guardan en audit_logs.action (VARCHAR(50)).
// Los módulos futuros (anulaciones, descuentos, cortesías) agregarán las suyas aquí.
const (
	ActionLoginSuccess = "LOGIN_SUCCESS"
	ActionLoginFailed  = "LOGIN_FAILED"
	ActionUserLocked   = "USER_LOCKED"
)

// Tipos de entidad a los que se refiere una entrada (audit_logs.entity_type).
const (
	EntityUser = "user"
)

// Límites de las columnas de audit_logs, para fallar con un error claro
// antes de que MySQL rechace o recorte el dato.
const (
	maxActionLen     = 50
	maxEntityTypeLen = 30
	maxIPLen         = 45
)

// ErrInvalidEntry se devuelve cuando una entrada no cumple las reglas básicas.
var ErrInvalidEntry = errors.New("entrada de auditoría inválida")

// Entry es una fila de audit_logs. La bitácora es inmutable: una entrada
// se crea una sola vez y nunca se modifica ni se borra.
//
// created_at no está aquí porque lo pone MySQL al insertar (DEFAULT CURRENT_TIMESTAMP),
// así la hora no depende del reloj de cada terminal.
type Entry struct {
	UserID     *int64         // nil cuando no se sabe quién fue (ej. código de empleado inexistente)
	Action     string         // una de las constantes Action*
	EntityType string         // ej. "user"; vacío si no aplica
	EntityID   *int64         // id de la entidad afectada; nil si no aplica
	Detail     map[string]any // datos extra; se guarda como JSON. NUNCA PINs ni datos de tarjeta
	IPAddress  string         // IP del cliente; vacía si no aplica
}

// Validate revisa que la entrada se pueda guardar.
func (e Entry) Validate() error {
	if e.Action == "" {
		return fmt.Errorf("%w: action es obligatoria", ErrInvalidEntry)
	}
	if len(e.Action) > maxActionLen {
		return fmt.Errorf("%w: action supera %d caracteres", ErrInvalidEntry, maxActionLen)
	}
	if len(e.EntityType) > maxEntityTypeLen {
		return fmt.Errorf("%w: entity_type supera %d caracteres", ErrInvalidEntry, maxEntityTypeLen)
	}
	if len(e.IPAddress) > maxIPLen {
		return fmt.Errorf("%w: ip_address supera %d caracteres", ErrInvalidEntry, maxIPLen)
	}
	return nil
}
