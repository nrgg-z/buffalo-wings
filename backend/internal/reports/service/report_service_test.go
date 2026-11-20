package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"buffalo-pos/internal/reports/domain"
)

type fakeReportRepo struct {
	lastDate string
}

func (f *fakeReportRepo) GetDailySummary(_ context.Context, date string) (*domain.DailySummary, error) {
	f.lastDate = date
	return &domain.DailySummary{Date: date}, nil
}

func (f *fakeReportRepo) ListVoids(_ context.Context, date string) ([]domain.VoidDetail, error) {
	f.lastDate = date
	return []domain.VoidDetail{}, nil
}

func TestReportServiceDates(t *testing.T) {
	repo := &fakeReportRepo{}
	svc := NewReportService(repo)
	svc.now = func() time.Time { return time.Date(2026, 11, 19, 15, 0, 0, 0, time.UTC) }

	// Fecha vacía usa el día UTC actual
	sum, err := svc.GetDailySummary(context.Background(), "")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if sum.Date != "2026-11-19" {
		t.Errorf("fecha = %s, esperaba 2026-11-19", sum.Date)
	}

	// Fecha inválida debe devolver ErrInvalidDate
	_, err = svc.GetDailySummary(context.Background(), "19-11-2026")
	if !errors.Is(err, domain.ErrInvalidDate) {
		t.Fatalf("esperaba ErrInvalidDate, obtuve %v", err)
	}

	// Fecha válida en ListVoids
	date, _, err := svc.ListVoids(context.Background(), "2026-11-18")
	if err != nil || date != "2026-11-18" || repo.lastDate != "2026-11-18" {
		t.Fatalf("esperaba fecha 2026-11-18 sin error, obtuve date=%s err=%v", date, err)
	}
}
