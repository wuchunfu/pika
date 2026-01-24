package migrate

import (
	"context"
	"strings"

	"github.com/dushixiang/pika/internal/migrate/v0_1_1"
	"github.com/dushixiang/pika/internal/migrate/v0_1_2"
	"github.com/dushixiang/pika/internal/migrate/v0_1_3"
	"github.com/dushixiang/pika/internal/service"
	"github.com/dushixiang/pika/pkg/version"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AutoMigrate 从开源版本升级到最新的版本
func AutoMigrate(logger *zap.Logger, db *gorm.DB, propertyService *service.PropertyService) error {
	ctx := context.Background()
	localVersion, _ := propertyService.GetSystemVersion(ctx)
	if localVersion == "" {
		localVersion = "0.1.0"
	}
	if strings.Compare(localVersion, "v0.1.0") < 0 {
		if err := v0_1_1.Migrate(logger, db); err != nil {
			return err
		}
	}
	// 升级到 v0.1.2 版本
	if strings.Compare(localVersion, "v0.1.2") < 0 {
		if err := v0_1_2.Migrate(logger, db); err != nil {
			return err
		}
	}
	// 升级到 v0.1.3 版本
	if strings.Compare(localVersion, "v0.1.3") < 0 {
		if err := v0_1_3.Migrate(logger, db); err != nil {
			return err
		}
	}

	return propertyService.SetSystemVersion(ctx, version.Version)
}
