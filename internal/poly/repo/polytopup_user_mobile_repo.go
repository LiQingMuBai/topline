package repo

import (
	"context"
	"ushield_bot/internal/poly/model"

	"gorm.io/gorm"
)

type PolytoupUserMobileRepository struct {
	db *gorm.DB
}

func NewPolytoupUserMobileRepository(db *gorm.DB) *PolytoupUserMobileRepository {
	return &PolytoupUserMobileRepository{
		db: db,
	}
}
func (r *PolytoupUserMobileRepository) ListAll(ctx context.Context, country_id, chat_id string) ([]model.UserMobile, error) {
	var pkgs []model.UserMobile
	err := r.db.WithContext(ctx).
		Model(&model.UserMobile{}).
		Select("id", "mobile,reminder_day").
		Where("status = 1 and country_id = ? and chat_id = ? ", country_id, chat_id).
		Scan(&pkgs).Error
	return pkgs, err

}
func (r *PolytoupUserMobileRepository) Query(ctx context.Context, ID string) (model.UserMobile, error) {
	var subscriptions []model.UserMobile
	err := r.db.WithContext(ctx).
		Model(&model.UserMobile{}).
		Select("id", "mobile,reminder_day,chat_id,country_id").
		Where("status = 1 and id = ?", ID).
		Scan(&subscriptions).Error
	return subscriptions[0], err

}

func (r *PolytoupUserMobileRepository) Get(ctx context.Context, id string) (model.UserMobile, error) {
	record := model.UserMobile{}
	err := r.db.Where(" status = 1 and id = ?  ", id).First(&record).Error
	return record, err
}

func (r *PolytoupUserMobileRepository) Count(ctx context.Context, countryID string, chatID int64) (count int64) {
	r.db.WithContext(ctx).Model(&model.UserMobile{}).Where("status = 1 and country_id = ? and chat_id = ?", countryID, chatID).Count(&count)
	return
}

// Create 创建新model
func (r *PolytoupUserMobileRepository) Create(ctx context.Context, pkg *model.UserMobile) error {
	return r.db.WithContext(ctx).Create(pkg).Error
}

// Update 更新model
func (r *PolytoupUserMobileRepository) Update(ctx context.Context, pkg *model.UserMobile) error {
	return r.db.WithContext(ctx).Save(pkg).Error
}

// Update 更新model
func (r *PolytoupUserMobileRepository) UpdateStatus(ctx context.Context, ID int64, _status int64) error {
	return r.db.WithContext(ctx).Model(&model.UserMobile{}).
		Where("id = ?", ID).
		Update("status", _status).Error
}

// Update 更新model
func (r *PolytoupUserMobileRepository) UpdateReminderDay(ctx context.Context, ID string, _day int64) error {
	return r.db.WithContext(ctx).Model(&model.UserMobile{}).
		Where("id = ?", ID).
		Update("reminder_day", _day).Error
}

// Delete 删除model
func (r *PolytoupUserMobileRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.UserMobile{}, id).Error
}

// Delete 删除model
func (r *PolytoupUserMobileRepository) Delete2(ctx context.Context, countryID, mobile string) error {
	return r.db.WithContext(ctx).Delete(&model.UserMobile{}, "country_id = ? AND mobile = ?", countryID, mobile).Error

}
