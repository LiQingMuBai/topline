package repo

import (
	"context"
	"errors"
	"ushield_bot/internal/poly/model"

	"gorm.io/gorm"
)

type ExpensesTopUpPlanRepo struct {
	db *gorm.DB
}

func NewExpensesTopUpPlanRepo(db *gorm.DB) *ExpensesTopUpPlanRepo {
	return &ExpensesTopUpPlanRepo{
		db: db,
	}
}
func (r *ExpensesTopUpPlanRepo) List(ctx context.Context) ([]model.ExpensesTopupPlan, error) {
	var planItmes []model.ExpensesTopupPlan
	err := r.db.WithContext(ctx).Model(&model.ExpensesTopupPlan{}).Scan(&planItmes).Error
	return planItmes, err
}
func (r *ExpensesTopUpPlanRepo) Get(ctx context.Context, id string) (model.ExpensesTopupPlan, error) {
	record := model.ExpensesTopupPlan{}
	err := r.db.Where(" id =? ", id).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 记录未找到，不是错误，只是表示不存在
		return record, nil // 第二个返回值表示是否存在
	}
	return record, err
}
