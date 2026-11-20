package mysqlrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"buffalo-pos/internal/audit/domain"
)

// AuditRepository guarda entradas en audit_logs.
//
// A propósito solo tiene un método de inserción: la bitácora es inmutable (RF-07),
// así que no existe forma de modificar ni borrar desde la aplicación. Además, la
// migración 0003 agrega triggers que lo impiden también desde la base de datos.
type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

const insertAuditQuery = `
	INSERT INTO audit_logs (user_id, action, entity_type, entity_id, detail, ip_address)
	VALUES (?, ?, ?, ?, ?, ?)`

// Record inserta una entrada en la bitácora.
func (r *AuditRepository) Record(ctx context.Context, e domain.Entry) error {
	if err := e.Validate(); err != nil {
		return err
	}

	// detail es JSON en MySQL; si no hay detalle se guarda NULL.
	var detail any
	if len(e.Detail) > 0 {
		b, err := json.Marshal(e.Detail)
		if err != nil {
			return fmt.Errorf("serializar detalle de auditoría: %w", err)
		}
		detail = string(b)
	}

	_, err := r.db.ExecContext(ctx, insertAuditQuery,
		e.UserID,
		e.Action,
		nullIfEmpty(e.EntityType),
		e.EntityID,
		detail,
		nullIfEmpty(e.IPAddress),
	)
	if err != nil {
		return fmt.Errorf("insertar entrada de auditoría: %w", err)
	}
	return nil
}

// nullIfEmpty convierte "" en NULL para no guardar textos vacíos en columnas opcionales.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
