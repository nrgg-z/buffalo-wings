// Package port define los contratos que el módulo de mesas necesita del exterior.
package port

import (
	"context"
	"errors"

	"buffalo-pos/internal/tables/domain"
)

var (
	// ErrTableNotFound: la mesa no existe (o está inactiva).
	ErrTableNotFound = errors.New("mesa no encontrada")
	// ErrVersionConflict: otra persona modificó la mesa (la versión ya no coincide).
	ErrVersionConflict = errors.New("conflicto de versión")
)

// TableRef identifica una mesa con la versión que el cliente vio.
type TableRef struct {
	ID      int64
	Version uint32
}

// TableRepository guarda y lee mesas. Todas las escrituras usan
// UPDATE ... WHERE id = ? AND version = ? y devuelven ErrVersionConflict
// si no afectan filas.
type TableRepository interface {
	List(ctx context.Context) ([]domain.Table, error)
	Assign(ctx context.Context, ref TableRef, guests int) error
	// Merge deja las mesas others combinadas a primary, todo o nada.
	Merge(ctx context.Context, primary TableRef, others []TableRef) error
	// Clean deja la mesa libre y libera las mesas que tenía combinadas.
	Clean(ctx context.Context, ref TableRef) error
}
