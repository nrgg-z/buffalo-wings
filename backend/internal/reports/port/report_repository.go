package port

import (
	"context"

	"buffalo-pos/internal/reports/domain"
)

type ReportRepository interface {
	GetDailySummary(ctx context.Context, date string) (*domain.DailySummary, error)
	ListVoids(ctx context.Context, date string) ([]domain.VoidDetail, error)
}
