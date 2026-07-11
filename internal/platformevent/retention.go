package platformevent

import (
	"context"
	"time"

	"github.com/LiteyukiStudio/devops/internal/model"
)

const (
	DefaultRetentionDays = 90
	DefaultCleanupBatch  = 1000
)

func DefaultRetentionCutoff(now time.Time) time.Time {
	return now.AddDate(0, 0, -DefaultRetentionDays)
}

func (s Service) CleanupBefore(ctx context.Context, before time.Time, batchSize int) (int64, error) {
	if s.DB == nil {
		return 0, nil
	}
	if batchSize <= 0 {
		batchSize = DefaultCleanupBatch
	}
	ids := s.DB.WithContext(ctx).
		Model(&model.PlatformEvent{}).
		Select("id").
		Where("occurred_at < ?", before).
		Order("occurred_at asc").
		Limit(batchSize)
	result := s.DB.WithContext(ctx).
		Where("id in (?)", ids).
		Delete(&model.PlatformEvent{})
	return result.RowsAffected, result.Error
}
