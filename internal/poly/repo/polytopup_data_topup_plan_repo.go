package repo

import (
	"context"
	"errors"
	"ushield_bot/internal/poly/model"

	"gorm.io/gorm"
)

type DataTopUpPlanRepo struct {
	db *gorm.DB
}

func NewDataTopUpPlanRepo(db *gorm.DB) *DataTopUpPlanRepo {
	return &DataTopUpPlanRepo{
		db: db,
	}
}
func (r *DataTopUpPlanRepo) List(ctx context.Context) ([]model.DataTopupPlan, error) {
	var planItmes []model.DataTopupPlan
	err := r.db.WithContext(ctx).Model(&model.DataTopupPlan{}).Scan(&planItmes).Error
	return planItmes, err
}
func (r *DataTopUpPlanRepo) Get(ctx context.Context, id string) (model.DataTopupPlan, error) {
	record := model.DataTopupPlan{}
	err := r.db.Where(" id =? ", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 记录未找到，不是错误，只是表示不存在
		return record, nil // 第二个返回值表示是否存在
	}
	return record, err
}
