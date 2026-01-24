package v0_1_3

import (
	"github.com/dushixiang/pika/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func Migrate(logger *zap.Logger, db *gorm.DB) error {
	logger.Info("开始执行 v0.1.3 版本数据迁移")

	migrator := db.Migrator()
	if migrator == nil {
		logger.Warn("无法获取数据库 migrator，跳过迁移")
		return nil
	}

	if !migrator.HasTable("agents") {
		logger.Info("未检测到 agents 表，跳过迁移")
		return nil
	}

	// 检查 weight 字段是否存在
	if !migrator.HasColumn("agents", "weight") {
		logger.Info("未检测到 weight 字段，跳过迁移")
		return nil
	}

	// 查询所有 weight 为 0 或 null 的 agents，按 name 升序排序
	var agents []models.Agent
	if err := db.Where("weight = ? OR weight IS NULL", 0).
		Order("name ASC").
		Find(&agents).Error; err != nil {
		logger.Error("查询 agents 失败", zap.Error(err))
		return err
	}

	if len(agents) == 0 {
		logger.Info("没有需要更新 weight 的探针，跳过迁移")
		return nil
	}

	logger.Info("找到需要更新 weight 的探针", zap.Int("count", len(agents)))

	// 按照 name 顺序分配 weight，从 1 开始递增
	// 这样按 name 字母顺序排在前面的探针，weight 值较小
	// 而在列表中按 weight 降序排列时，name 靠后的会显示在前面
	for i, agent := range agents {
		weight := i + 1
		if err := db.Model(&models.Agent{}).
			Where("id = ?", agent.ID).
			Update("weight", weight).Error; err != nil {
			logger.Error("更新探针 weight 失败",
				zap.String("id", agent.ID),
				zap.String("name", agent.Name),
				zap.Int("weight", weight),
				zap.Error(err))
			return err
		}
		logger.Debug("已更新探针 weight",
			zap.String("id", agent.ID),
			zap.String("name", agent.Name),
			zap.Int("weight", weight))
	}

	logger.Info("v0.1.3 版本数据迁移完成", zap.Int("updated", len(agents)))
	return nil
}
