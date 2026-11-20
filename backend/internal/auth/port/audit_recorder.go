package port

import (
	"context"

	auditdomain "buffalo-pos/internal/audit/domain"
)

// AuditRecorder es lo que el servicio de autenticación necesita de la bitácora.
//
// La interfaz se declara aquí (en quien la usa) y no en el módulo audit: es la
// forma idiomática en Go. Así auth no depende de MySQL, y en las pruebas se
// reemplaza por un falso que guarda las entradas en memoria.
type AuditRecorder interface {
	Record(ctx context.Context, e auditdomain.Entry) error
}
