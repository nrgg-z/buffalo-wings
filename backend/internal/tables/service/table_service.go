// Package service contiene los casos de uso de las mesas.
package service

import (
	"context"

	"buffalo-pos/internal/tables/domain"
	"buffalo-pos/internal/tables/port"
)

type TableService struct {
	repo port.TableRepository
}

func NewTableService(repo port.TableRepository) *TableService {
	return &TableService{repo: repo}
}

// List devuelve el mapa de mesas.
func (s *TableService) List(ctx context.Context) ([]domain.Table, error) {
	tables, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	domain.FillEffectiveCapacity(tables)
	return tables, nil
}

// Assign sienta comensales en una mesa libre.
func (s *TableService) Assign(ctx context.Context, ref port.TableRef, guests int) (*domain.Table, error) {
	tables, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	t, ok := find(tables, ref.ID)
	if !ok {
		return nil, port.ErrTableNotFound
	}
	if err := t.CanAssign(guests); err != nil {
		return nil, err
	}
	if err := s.repo.Assign(ctx, ref, guests); err != nil {
		return nil, err
	}
	return s.get(ctx, ref.ID)
}

// Merge combina mesas libres, de la misma zona y con códigos consecutivos.
// Devuelve la mesa principal.
func (s *TableService) Merge(ctx context.Context, refs []port.TableRef) (*domain.Table, error) {
	seen := map[int64]bool{}
	for _, r := range refs {
		if seen[r.ID] {
			return nil, domain.ErrInvalidMerge
		}
		seen[r.ID] = true
	}

	tables, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	selected := make([]domain.Table, 0, len(refs))
	versions := map[int64]uint32{}
	for _, r := range refs {
		t, ok := find(tables, r.ID)
		if !ok {
			return nil, port.ErrTableNotFound
		}
		selected = append(selected, t)
		versions[r.ID] = r.Version
	}

	primary, others, err := domain.PlanMerge(selected)
	if err != nil {
		return nil, err
	}
	otherRefs := make([]port.TableRef, 0, len(others))
	for _, o := range others {
		otherRefs = append(otherRefs, port.TableRef{ID: o.ID, Version: versions[o.ID]})
	}
	if err := s.repo.Merge(ctx, port.TableRef{ID: primary.ID, Version: versions[primary.ID]}, otherRefs); err != nil {
		return nil, err
	}
	return s.get(ctx, primary.ID)
}

// MarkClean deja libre una mesa que estaba por limpiar.
func (s *TableService) MarkClean(ctx context.Context, ref port.TableRef) (*domain.Table, error) {
	tables, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	t, ok := find(tables, ref.ID)
	if !ok {
		return nil, port.ErrTableNotFound
	}
	if err := t.CanClean(); err != nil {
		return nil, err
	}
	if err := s.repo.Clean(ctx, ref); err != nil {
		return nil, err
	}
	return s.get(ctx, ref.ID)
}

func (s *TableService) get(ctx context.Context, id int64) (*domain.Table, error) {
	tables, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	t, ok := find(tables, id)
	if !ok {
		return nil, port.ErrTableNotFound
	}
	return &t, nil
}

func find(tables []domain.Table, id int64) (domain.Table, bool) {
	for _, t := range tables {
		if t.ID == id {
			return t, true
		}
	}
	return domain.Table{}, false
}
