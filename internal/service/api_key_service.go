package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/dushixiang/pika/internal/models"
	"github.com/dushixiang/pika/internal/repo"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ApiKeyService struct {
	logger     *zap.Logger
	ApiKeyRepo *repo.ApiKeyRepo
}

func NewApiKeyService(logger *zap.Logger, db *gorm.DB) *ApiKeyService {
	return &ApiKeyService{
		logger:     logger,
		ApiKeyRepo: repo.NewApiKeyRepo(db),
	}
}

// GenerateApiKey 生成API密钥
func (s *ApiKeyService) GenerateApiKey(ctx context.Context, name, userID, keyType string) (*models.ApiKey, error) {
	// 生成32字节随机密钥
	key, err := s.generateSecureKey(32)
	if err != nil {
		return nil, err
	}

	if keyType == "" {
		keyType = "agent"
	}

	now := time.Now().UnixMilli()
	apiKey := &models.ApiKey{
		ID:        uuid.NewString(),
		Name:      name,
		Key:       key,
		Type:      keyType,
		Enabled:   true,
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.ApiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, err
	}

	s.logger.Info("api key generated",
		zap.String("keyID", apiKey.ID),
		zap.String("name", name),
		zap.String("userID", userID))

	return apiKey, nil
}

// ValidateApiKey 验证API密钥
func (s *ApiKeyService) ValidateApiKey(ctx context.Context, key string, keyType string) (*models.ApiKey, error) {
	if key == "" {
		return nil, errors.New("api key is required")
	}

	apiKey, err := s.ApiKeyRepo.FindEnabledByKey(ctx, key)
	if err != nil {
		s.logger.Warn("invalid api key", zap.String("key", maskSecret(key)))
		return nil, errors.New("invalid api key")
	}

	// 兼容旧版本没有 type 字段值的通信密钥。
	if apiKey.Type == "" {
		apiKey.Type = "agent"
	}

	// 验证密钥类型匹配
	if keyType != "" && apiKey.Type != keyType {
		s.logger.Warn("api key type mismatch",
			zap.String("keyID", apiKey.ID),
			zap.String("expected", keyType),
			zap.String("actual", apiKey.Type))
		return nil, errors.New("invalid api key")
	}

	return apiKey, nil
}

// GetApiKey 获取API密钥信息
func (s *ApiKeyService) GetApiKey(ctx context.Context, id string) (*models.ApiKey, error) {
	apiKey, err := s.ApiKeyRepo.FindById(ctx, id)
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// ListApiKeys 列出所有API密钥
func (s *ApiKeyService) ListApiKeys(ctx context.Context, page, pageSize int) ([]models.ApiKey, int64, error) {
	return s.ApiKeyRepo.ListAll(ctx, page, pageSize)
}

// ListUserApiKeys 列出用户的API密钥
func (s *ApiKeyService) ListUserApiKeys(ctx context.Context, userID string, page, pageSize int) ([]models.ApiKey, int64, error) {
	return s.ApiKeyRepo.ListByUser(ctx, userID, page, pageSize)
}

// UpdateApiKeyName 更新API密钥名称
func (s *ApiKeyService) UpdateApiKeyName(ctx context.Context, id, name string) error {
	if err := s.ApiKeyRepo.UpdateName(ctx, id, name); err != nil {
		return err
	}

	s.logger.Info("api key name updated",
		zap.String("keyID", id),
		zap.String("name", name))

	return nil
}

// EnableApiKey 启用API密钥
func (s *ApiKeyService) EnableApiKey(ctx context.Context, id string) error {
	if err := s.ApiKeyRepo.UpdateEnabled(ctx, id, true); err != nil {
		return err
	}

	s.logger.Info("api key enabled", zap.String("keyID", id))
	return nil
}

// DisableApiKey 禁用API密钥
func (s *ApiKeyService) DisableApiKey(ctx context.Context, id string) error {
	if err := s.ApiKeyRepo.UpdateEnabled(ctx, id, false); err != nil {
		return err
	}

	s.logger.Info("api key disabled", zap.String("keyID", id))
	return nil
}

// DeleteApiKey 删除API密钥
func (s *ApiKeyService) DeleteApiKey(ctx context.Context, id string) error {
	if err := s.ApiKeyRepo.DeleteById(ctx, id); err != nil {
		return err
	}

	s.logger.Info("api key deleted", zap.String("keyID", id))
	return nil
}

// FillLegacyApiKeyType 将旧版本未标记类型的密钥回填为通信密钥。
func (s *ApiKeyService) FillLegacyApiKeyType(ctx context.Context) error {
	if err := s.ApiKeyRepo.FillEmptyType(ctx, "agent"); err != nil {
		return err
	}
	s.logger.Info("legacy api key type filled")
	return nil
}

// generateSecureKey 生成安全的随机密钥
func (s *ApiKeyService) generateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "..." + secret[len(secret)-4:]
}
