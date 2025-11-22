package repo

import (
	"context"

	"github.com/dushixiang/pika/internal/models"
	"github.com/go-orz/orz"
	"gorm.io/gorm"
)

type MonitorStatsRepo struct {
	orz.Repository[models.MonitorStats, string]
	db *gorm.DB
}

func NewMonitorStatsRepo(db *gorm.DB) *MonitorStatsRepo {
	return &MonitorStatsRepo{
		Repository: orz.NewRepository[models.MonitorStats, string](db),
		db:         db,
	}
}

func (r *MonitorStatsRepo) FindByMonitorId(ctx context.Context, monitorId string) ([]models.MonitorStats, error) {
	var statsList []models.MonitorStats
	err := r.db.WithContext(ctx).
		Where("monitor_id = ?", monitorId).
		Find(&statsList).Error
	return statsList, err
}

func (r *MonitorStatsRepo) FindByMonitorIdIn(ctx context.Context, monitorIds []string) ([]models.MonitorStats, error) {
	var statsList []models.MonitorStats
	err := r.db.WithContext(ctx).
		Where("monitor_id in ?", monitorIds).
		Find(&statsList).Error
	return statsList, err
}

func (r *MonitorStatsRepo) DeleteByMonitorId(ctx context.Context, monitorId string) error {
	return r.db.WithContext(ctx).
		Where("monitor_id = ?", monitorId).
		Delete(&models.MonitorStats{}).Error
}
