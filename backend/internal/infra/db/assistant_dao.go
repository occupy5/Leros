package db

import (
	"context"

	"gorm.io/gorm"

	"github.com/insmtx/Leros/backend/types"
)

// GetAssistantsByIDs 批量根据ID查询数字助手
func GetAssistantsByIDs(ctx context.Context, db *gorm.DB, ids []uint) ([]*types.DigitalAssistant, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var entities []*types.DigitalAssistant
	err := db.WithContext(ctx).Where("id IN (?)", ids).Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}
