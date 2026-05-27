package db

import (
	"context"

	"gorm.io/gorm"

	"github.com/insmtx/Leros/backend/types"
)

// GetUsersByIDs 批量根据用户ID查询用户信息
func GetUsersByIDs(ctx context.Context, db *gorm.DB, ids []uint) ([]*types.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var entities []*types.User
	err := db.WithContext(ctx).Where("id IN (?)", ids).Find(&entities).Error
	if err != nil {
		return nil, err
	}
	return entities, nil
}
