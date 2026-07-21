package repositories

import (
	"context"
	"ushield_bot/internal/request"

	_ "github.com/go-sql-driver/mysql"

	"ushield_bot/internal/domain"

	"gorm.io/gorm"
)

type UserUSDTDepositsRepository struct {
	db *gorm.DB
}

func NewUserUSDTDepositRepository(db *gorm.DB) *UserUSDTDepositsRepository {
	return &UserUSDTDepositsRepository{
		db: db,
	}
}

func (r *UserUSDTDepositsRepository) Create(ctx context.Context, deposit *domain.UserUSDTDeposits) error {
	return r.db.WithContext(ctx).Create(deposit).Error
}

func (r *UserUSDTDepositsRepository) ListByUserAndStatus(ctx context.Context, chatID int64, status int64) ([]domain.UserUSDTDeposits, error) {
	var deposits []domain.UserUSDTDeposits
	err := r.db.Select("id,amount,order_no, DATE_FORMAT(created_at, '%m-%d') as created_date").
		Where("user_id = ?", chatID).
		Where("status = ?", status).
		Find(&deposits).Error
	return deposits, err

}
func (r *UserUSDTDepositsRepository) ListPageByUser(ctx context.Context, info request.UserUsdtDepositsSearch, chatID int64) (list []domain.UserUSDTDeposits, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)
	db := r.db.Model(&domain.UserUSDTDeposits{}).
		Select("id,amount,order_no, DATE_FORMAT(created_at, '%m-%d') as created_date").
		Where("user_id = ?", chatID).
		Where("status = ?", 1)
	var deposits []domain.UserUSDTDeposits

	err = db.Count(&total).Error
	if err != nil {
		return
	}

	if limit != 0 {
		db = db.Limit(int(limit)).Offset(int(offset)).Order("id DESC")
	}

	err = db.Find(&deposits).Error
	return deposits, total, err
}

func (r *UserUSDTDepositsRepository) GetByOrderNo(ctx context.Context, orderNo string) (domain.UserUSDTDeposits, error) {
	var deposit domain.UserUSDTDeposits
	err := r.db.WithContext(ctx).
		Find(&deposit, "order_no = ?", orderNo).Error
	return deposit, err
}

func (r *UserUSDTDepositsRepository) UpdateStatusByID(ctx context.Context, id int64, status int64) error {
	return r.db.WithContext(ctx).Model(&domain.UserUSDTDeposits{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *UserUSDTDepositsRepository) UpdateStatusByOrderNo(ctx context.Context, orderNo string, status int64) error {
	return r.db.WithContext(ctx).Model(&domain.UserUSDTDeposits{}).
		Where("order_no = ?", orderNo).
		Update("status", status).Error
}
