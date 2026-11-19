// Package domain define las reglas de negocio puras de las mesas (sin SQL ni HTTP).
package domain

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Status es el estado de una mesa. La base y el dominio usan inglés;
// la API los traduce con APIName.
type Status string

const (
	StatusFree     Status = "FREE"
	StatusOccupied Status = "OCCUPIED"
	StatusReserved Status = "RESERVED"
	StatusCleaning Status = "CLEANING"
)

// APIName devuelve el estado como lo ve el cliente de la API.
func (s Status) APIName() string {
	switch s {
	case StatusFree:
		return "LIBRE"
	case StatusOccupied:
		return "OCUPADA"
	case StatusReserved:
		return "RESERVADA"
	case StatusCleaning:
		return "POR_LIMPIAR"
	}
	return string(s)
}

// Errores de reglas de negocio.
var (
	ErrInvalidGuests   = errors.New("cantidad de comensales inválida")
	ErrOverCapacity    = errors.New("los comensales superan la capacidad de la mesa")
	ErrTableNotFree    = errors.New("la mesa no está libre")
	ErrTableMerged     = errors.New("la mesa está combinada con otra")
	ErrNotCleaning     = errors.New("la mesa no está por limpiar")
	ErrInvalidMerge    = errors.New("petición de combinación inválida")
	ErrMergeNotAllowed = errors.New("las mesas no se pueden combinar")
)

// Table es una mesa del restaurante.
type Table struct {
	ID                int64
	Code              string
	Zone              string
	Capacity          int
	EffectiveCapacity int // capacidad propia más la de las mesas combinadas a ella
	Status            Status
	CurrentGuests     int
	AssignedWaiterID  *int64
	MergedIntoID      *int64
	Version           uint32
}

// FillEffectiveCapacity calcula EffectiveCapacity de cada mesa.
func FillEffectiveCapacity(tables []Table) {
	extra := map[int64]int{}
	for _, t := range tables {
		if t.MergedIntoID != nil {
			extra[*t.MergedIntoID] += t.Capacity
		}
	}
	for i := range tables {
		tables[i].EffectiveCapacity = tables[i].Capacity + extra[tables[i].ID]
	}
}

// CanAssign revisa que se puedan sentar guests comensales en la mesa.
func (t Table) CanAssign(guests int) error {
	if guests < 1 {
		return ErrInvalidGuests
	}
	if t.MergedIntoID != nil {
		return ErrTableMerged
	}
	if t.Status != StatusFree {
		return ErrTableNotFree
	}
	if guests > t.EffectiveCapacity {
		return ErrOverCapacity
	}
	return nil
}

// CanClean revisa que la mesa se pueda marcar como limpia (solo desde CLEANING).
func (t Table) CanClean() error {
	if t.Status != StatusCleaning {
		return ErrNotCleaning
	}
	return nil
}

// PlanMerge valida que las mesas se puedan combinar y devuelve la principal
// (la de código más bajo) y las demás. Reglas: al menos 2 mesas, todas libres
// y sin combinar, de la misma zona y con códigos consecutivos (M-04, M-05...).
func PlanMerge(tables []Table) (primary Table, others []Table, err error) {
	if len(tables) < 2 {
		return Table{}, nil, fmt.Errorf("%w: se requieren al menos 2 mesas", ErrInvalidMerge)
	}

	type numbered struct {
		t   Table
		num int
	}
	list := make([]numbered, 0, len(tables))
	prefix := ""
	for i, t := range tables {
		if t.Status != StatusFree || t.MergedIntoID != nil {
			return Table{}, nil, fmt.Errorf("%w: %s no está libre", ErrMergeNotAllowed, t.Code)
		}
		if t.Zone != tables[0].Zone {
			return Table{}, nil, fmt.Errorf("%w: las mesas son de distinta zona", ErrMergeNotAllowed)
		}
		p, n, ok := splitCode(t.Code)
		if !ok {
			return Table{}, nil, fmt.Errorf("%w: código %q no reconocido", ErrMergeNotAllowed, t.Code)
		}
		if i == 0 {
			prefix = p
		} else if p != prefix {
			return Table{}, nil, fmt.Errorf("%w: códigos no consecutivos", ErrMergeNotAllowed)
		}
		list = append(list, numbered{t, n})
	}

	sort.Slice(list, func(i, j int) bool { return list[i].num < list[j].num })
	for i := 1; i < len(list); i++ {
		if list[i].num != list[i-1].num+1 {
			return Table{}, nil, fmt.Errorf("%w: códigos no consecutivos", ErrMergeNotAllowed)
		}
	}

	for _, n := range list[1:] {
		others = append(others, n.t)
	}
	return list[0].t, others, nil
}

// splitCode separa "M-04" en el prefijo "M-" y el número 4.
func splitCode(code string) (prefix string, num int, ok bool) {
	i := strings.LastIndex(code, "-")
	if i < 0 {
		return "", 0, false
	}
	n, err := strconv.Atoi(code[i+1:])
	if err != nil {
		return "", 0, false
	}
	return code[:i+1], n, true
}
