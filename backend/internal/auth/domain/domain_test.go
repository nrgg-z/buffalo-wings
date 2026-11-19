package domain

import (
	"testing"
	"time"
)

func TestHasAnyRoleAceptaRolPermitido(t *testing.T) {
	p := Principal{Role: RoleWaiter}
	if !p.HasAnyRole(RoleHostess, RoleWaiter) {
		t.Error("WAITER debía pasar cuando está en la lista")
	}
}

func TestHasAnyRoleRechazaRolNoPermitido(t *testing.T) {
	p := Principal{Role: RoleKitchen}
	if p.HasAnyRole(RoleCashier, RoleManager) {
		t.Error("KITCHEN no debía pasar")
	}
}

func TestHasAnyRoleAdminSiemprePasa(t *testing.T) {
	p := Principal{Role: RoleAdmin}
	if !p.HasAnyRole(RoleCashier) {
		t.Error("ADMIN debía pasar aunque no esté en la lista")
	}
}

func TestHasAnyRoleManagerNoEsImplicito(t *testing.T) {
	p := Principal{Role: RoleManager}
	if p.HasAnyRole(RoleKitchen) {
		t.Error("MANAGER no debía pasar si no está en la lista")
	}
}

func TestRegisterFailedAttemptBloqueaAlLlegarAlMaximo(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	u := &User{}

	u.RegisterFailedAttempt(now, 3, 5*time.Minute)
	u.RegisterFailedAttempt(now, 3, 5*time.Minute)
	if u.IsLocked(now) {
		t.Fatal("con 2 intentos no debía bloquearse")
	}

	u.RegisterFailedAttempt(now, 3, 5*time.Minute)
	if !u.IsLocked(now) {
		t.Fatal("con 3 intentos debía bloquearse")
	}
	if u.IsLocked(now.Add(5 * time.Minute)) {
		t.Error("a los 5 minutos exactos el bloqueo debía haber vencido")
	}
}

func TestRegisterFailedAttemptReiniciaTrasBloqueoVencido(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	until := now.Add(-time.Second)
	u := &User{FailedAttempts: 3, LockedUntil: &until}

	u.RegisterFailedAttempt(now, 3, 5*time.Minute)

	if u.FailedAttempts != 1 || u.LockedUntil != nil {
		t.Errorf("esperaba 1 intento y sin bloqueo, llegó %d intentos y bloqueo %v", u.FailedAttempts, u.LockedUntil)
	}
}
