package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestEntryValidaAceptaEntradaCorrecta(t *testing.T) {
	e := Entry{Action: ActionLoginSuccess, EntityType: EntityUser, IPAddress: "127.0.0.1"}
	if err := e.Validate(); err != nil {
		t.Errorf("no se esperaba error: %v", err)
	}
}

func TestEntryValidaRechazaSinAccion(t *testing.T) {
	if err := (Entry{}).Validate(); !errors.Is(err, ErrInvalidEntry) {
		t.Errorf("esperaba ErrInvalidEntry, llegó %v", err)
	}
}

func TestEntryValidaRechazaAccionMuyLarga(t *testing.T) {
	e := Entry{Action: strings.Repeat("A", 51)}
	if err := e.Validate(); !errors.Is(err, ErrInvalidEntry) {
		t.Errorf("esperaba ErrInvalidEntry, llegó %v", err)
	}
}
