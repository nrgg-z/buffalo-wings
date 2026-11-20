package domain

import (
	"testing"
	"time"
)

func TestComputeTiming(t *testing.T) {
	now := time.Date(2026, 11, 19, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		ago          time.Duration
		wantMinutes  int
		wantAlertLvl string
	}{
		{"recién creado es GREEN", 3 * time.Minute, 3, "GREEN"},
		{"10 minutos es YELLOW", 10 * time.Minute, 10, "YELLOW"},
		{"15 minutos es RED", 15 * time.Minute, 15, "RED"},
		{"fecha futura se acota a 0", -2 * time.Minute, 0, "GREEN"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item := KDSItem{CreatedAt: now.Add(-tc.ago)}
			item.ComputeTiming(now)
			if item.ElapsedMinutes != tc.wantMinutes || item.AlertLevel != tc.wantAlertLvl {
				t.Fatalf("got (%d, %s), want (%d, %s)",
					item.ElapsedMinutes, item.AlertLevel, tc.wantMinutes, tc.wantAlertLvl)
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	valid := [][2]string{
		{"PENDING", "IN_PROGRESS"},
		{"PENDING", "READY"},
		{"IN_PROGRESS", "READY"},
		{"READY", "DELIVERED"},
	}
	for _, pair := range valid {
		if !IsValidTransition(pair[0], pair[1]) {
			t.Errorf("esperaba transición válida %s -> %s", pair[0], pair[1])
		}
	}

	invalid := [][2]string{
		{"PENDING", "DELIVERED"},
		{"IN_PROGRESS", "PENDING"},
		{"READY", "IN_PROGRESS"},
		{"DELIVERED", "READY"},
	}
	for _, pair := range invalid {
		if IsValidTransition(pair[0], pair[1]) {
			t.Errorf("esperaba transición inválida %s -> %s", pair[0], pair[1])
		}
	}
}
