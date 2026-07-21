package repositories

import (
	"context"
	"gorm.io/gorm"
	"ushield_bot/internal/domain"
)

type UserUSDTPlaceholdersRepository struct {
	db *gorm.DB
}

func NewUserUSDTPlaceholdersRepository(db *gorm.DB) *UserUSDTPlaceholdersRepository {
	return &UserUSDTPlaceholdersRepository{
		db: db,
	}
}
func (r *UserUSDTPlaceholdersRepository) ListAll(ctx context.Context) ([]domain.UserUsdtPlaceholders, error) {
	var placeholders []domain.UserUsdtPlaceholders
	err := r.db.WithContext(ctx).
		Model(&domain.UserUsdtPlaceholders{}).
		Select("id", "placeholder").
		Where("status = ?", 0).
		Scan(&placeholders).Error
	return placeholders, err

}

func (r *UserUSDTPlaceholdersRepository) UpdateStatusByPlaceholder(ctx context.Context, placeholder string, status int64) error {
	return r.db.WithContext(ctx).Model(&domain.UserUsdtPlaceholders{}).
		Where("placeholder = ?", placeholder).
		Update("status", status).Error
}

func (r *UserUSDTPlaceholdersRepository) UpdateStatusByID(ctx context.Context, id int64, status int64) error {
	return r.db.WithContext(ctx).Model(&domain.UserUsdtPlaceholders{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *UserUSDTPlaceholdersRepository) GetRandomAvailable(ctx context.Context) (domain.UserUsdtPlaceholders, error) {
	var placeholders domain.UserUsdtPlaceholders
	err := r.db.WithContext(ctx).Order("RAND()").
		Find(&placeholders, "status = ?", 0).Error
	return placeholders, err
}
