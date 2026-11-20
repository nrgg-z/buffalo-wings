package service

import (
	"context"
	"strings"
	"time"

	"buffalo-pos/internal/reports/domain"
	"buffalo-pos/internal/reports/port"
)

type ReportService struct {
	repo port.ReportRepository
	now  func() time.Time
}

func NewReportService(repo port.ReportRepository) *ReportService {
	return &ReportService{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

func (s *ReportService) GetDailySummary(ctx context.Context, rawDate string) (*domain.DailySummary, error) {
	date, err := s.normalizeDate(rawDate)
	if err != nil {
		return nil, err
	}
	return s.repo.GetDailySummary(ctx, date)
}

func (s *ReportService) ListVoids(ctx context.Context, rawDate string) (string, []domain.VoidDetail, error) {
	date, err := s.normalizeDate(rawDate)
	if err != nil {
		return "", nil, err
	}
	items, err := s.repo.ListVoids(ctx, date)
	if err != nil {
		return "", nil, err
	}
	return date, items, nil
}

func (s *ReportService) normalizeDate(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return s.now().Format("2006-01-02"), nil
	}
	if _, err := time.Parse("2006-01-02", raw); err != nil {
		return "", domain.ErrInvalidDate
	}
	return raw, nil
}
