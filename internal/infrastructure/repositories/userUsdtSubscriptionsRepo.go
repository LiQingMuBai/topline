package repositories

import (
	"context"
	"gorm.io/gorm"
	"ushield_bot/internal/domain"
)

type UserUSDTSubscriptionsRepository struct {
	db *gorm.DB
}

func NewUserUSDTSubscriptionsRepository(db *gorm.DB) *UserUSDTSubscriptionsRepository {
	return &UserUSDTSubscriptionsRepository{
		db: db,
	}
}

func (r *UserUSDTSubscriptionsRepository) ListAvailable(ctx context.Context) ([]domain.UserUsdtSubscriptions, error) {
	var subscriptions []domain.UserUsdtSubscriptions
	err := r.db.WithContext(ctx).
		Model(&domain.UserUsdtSubscriptions{}).
		Select("id", "name", "amount").
		Where("status = ?", 0).
		Scan(&subscriptions).Error
	return subscriptions, err

}
