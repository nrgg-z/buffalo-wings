package domain

import (
	"errors"
	"testing"
)

func tbl(id int64, code, zone string, st Status) Table {
	return Table{ID: id, Code: code, Zone: zone, Capacity: 4, Status: st}
}

func TestStatusAPIName(t *testing.T) {
	cases := map[Status]string{
		StatusFree: "LIBRE", StatusOccupied: "OCUPADA",
		StatusReserved: "RESERVADA", StatusCleaning: "POR_LIMPIAR",
	}
	for s, want := range cases {
		if got := s.APIName(); got != want {
			t.Errorf("%s: got %s, want %s", s, got, want)
		}
	}
}

func TestPlanMerge(t *testing.T) {
	occupied := tbl(5, "M-05", "SALON_CENTRAL", StatusOccupied)
	tests := []struct {
		name    string
		tables  []Table
		wantErr error
		primary string
	}{
		{"principal es primera seleccionada", []Table{tbl(5, "M-05", "Z", StatusFree), tbl(4, "M-04", "Z", StatusFree)}, nil, "M-05"},
		{"una sola", []Table{tbl(4, "M-04", "Z", StatusFree)}, ErrInvalidMerge, ""},
		{"no libre", []Table{tbl(4, "M-04", "SALON_CENTRAL", StatusFree), occupied}, ErrMergeNotAllowed, ""},
		{"otra zona", []Table{tbl(3, "M-03", "BARRA", StatusFree), tbl(4, "M-04", "SALON_CENTRAL", StatusFree)}, ErrMergeNotAllowed, ""},
		{"saltan código", []Table{tbl(4, "M-04", "Z", StatusFree), tbl(6, "M-06", "Z", StatusFree)}, ErrMergeNotAllowed, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, others, err := PlanMerge(tc.tables)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if err == nil && (p.Code != tc.primary || len(others) != len(tc.tables)-1) {
				t.Errorf("primary = %s, others = %d", p.Code, len(others))
			}
		})
	}
}

func TestCanAssignAndClean(t *testing.T) {
	free := Table{Status: StatusFree, EffectiveCapacity: 4}
	if err := free.CanAssign(0); !errors.Is(err, ErrInvalidGuests) {
		t.Errorf("0 comensales: %v", err)
	}
	if err := free.CanAssign(5); !errors.Is(err, ErrOverCapacity) {
		t.Errorf("sobrecupo: %v", err)
	}
	if err := free.CanAssign(4); err != nil {
		t.Errorf("válido: %v", err)
	}
	if err := (Table{Status: StatusOccupied}).CanAssign(1); !errors.Is(err, ErrTableNotFree) {
		t.Errorf("ocupada: %v", err)
	}
	if err := (Table{Status: StatusFree}).CanClean(); !errors.Is(err, ErrNotCleaning) {
		t.Errorf("limpiar libre: %v", err)
	}
	if err := (Table{Status: StatusCleaning}).CanClean(); err != nil {
		t.Errorf("limpiar: %v", err)
	}
}
