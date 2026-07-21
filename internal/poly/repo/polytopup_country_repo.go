package repo

import (
	"context"
	"errors"
	"ushield_bot/internal/poly/model"

	"gorm.io/gorm"
)

type CountryRepo struct {
	db *gorm.DB
}

func NewCountryRepo(db *gorm.DB) *CountryRepo {
	return &CountryRepo{
		db: db,
	}
}
func (r *CountryRepo) List(ctx context.Context) ([]model.Country, error) {
	var countryItems []model.Country
	err := r.db.WithContext(ctx).Model(&model.Country{}).Scan(&countryItems).Error
	return countryItems, err
}

func (r *CountryRepo) Get(ctx context.Context, id string) (model.Country, error) {
	record := model.Country{}
	err := r.db.Where(" id =? ", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 记录未找到，不是错误，只是表示不存在
		return record, nil // 第二个返回值表示是否存在
	}
	return record, err
}
