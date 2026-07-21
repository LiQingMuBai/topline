package repositories

import (
	"context"
	"gorm.io/gorm"
	"ushield_bot/internal/domain"
)

type SystemUserRepository struct {
	db *gorm.DB
}

func NewSystemUserRepository(db *gorm.DB) *SystemUserRepository {
	return &SystemUserRepository{
		db: db,
	}
}

func (r *SystemUserRepository) GetAddressesByUsername(ctx context.Context, username string) (address, depositAddress string, err error) {
	var sysUser domain.SysUser
	result := r.db.WithContext(ctx).
		Model(&domain.SysUser{}).
		Select("address, deposit_address").
		Where("username = ?", username).
		First(&sysUser)
	if result.Error != nil {
		return "", "", result.Error
	}

	return sysUser.Address, sysUser.DepositAddress, nil
}
