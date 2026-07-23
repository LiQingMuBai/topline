package repositories

import (
	"context"

	_ "github.com/go-sql-driver/mysql"

	"ushield_bot/internal/domain"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Save 使用 GORM Save 持久化完整用户对象。
func (r *UserRepository) Save(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *UserRepository) UpdateBackupChat(ctx context.Context, backup string, _associates int64) error {
	query := "UPDATE tg_users SET backup_chat_id = ?  WHERE associates = ?"
	tx := r.db.Exec(query, backup, _associates)
	return tx.Error
}

// Create 使用 GORM 创建用户，作为当前活跃链路的统一创建入口。
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) UpdateUsernameByChatID(username string, chatID int64) error {
	query := "UPDATE tg_users SET username = ? WHERE associates = ?"
	tx := r.db.Exec(query, username, chatID)
	return tx.Error
}

func (r *UserRepository) Update(user domain.User) error {
	query := "UPDATE tg_users SET associates = $1, tron_amount = $2 WHERE username = $3"
	tx := r.db.Exec(query, user.Associates, user.TronAmount, user.Username)
	return tx.Error
}

func (r *UserRepository) UpdateAddress(user domain.User) error {
	query := "UPDATE tg_users SET address = ? , private_key = ?  WHERE id = ?"
	tx := r.db.Exec(query, user.Address, user.Key, user.Id)
	return tx.Error
}

func (r *UserRepository) UpdateTimes(_times uint64, _username string) error {
	query := "UPDATE tg_users SET times = ?  WHERE username = ?"
	tx := r.db.Exec(query, _times, _username)
	return tx.Error
}
func (r *UserRepository) UpdateBundleTimes(_bundleTimes int64, _chatID int64) error {
	query := "UPDATE tg_users SET bundle_times = ?  WHERE associates = ?"
	tx := r.db.Exec(query, _bundleTimes, _chatID)
	return tx.Error
}

func (r *UserRepository) UpdateSTTimes(_bundleTimes int64, _chatID int64) error {
	query := "UPDATE tg_users SET st_times = ?  WHERE associates = ?"
	tx := r.db.Exec(query, _bundleTimes, _chatID)
	return tx.Error
}

func (r *UserRepository) UpdateTrxAmount(_trxAmount string, _chatID int64) error {
	query := "UPDATE tg_users SET tron_amount = ?  WHERE associates = ?"
	tx := r.db.Exec(query, _trxAmount, _chatID)
	return tx.Error
}

func (r *UserRepository) UpdateUSDTAmount(_amount string, _chatID int64) error {
	query := "UPDATE tg_users SET amount = ?  WHERE associates = ?"
	tx := r.db.Exec(query, _amount, _chatID)
	return tx.Error
}

func (r *UserRepository) UpdateTimesByChatID(_times uint64, _chatID int64) error {
	query := "UPDATE tg_users SET times = ?  WHERE associates = ?"
	tx := r.db.Exec(query, _times, _chatID)
	return tx.Error
}

//associates VARCHAR(255),
//amount VARCHAR(255) ,
//tron_amount VARCHAR(255),
//tron_address VARCHAR(50),
//eth_address VARCHAR(50),
//eth_amount VARCHAR(255),

func (r *UserRepository) GetByUsername(_username string) (domain.User, error) {

	jason := domain.User{}

	err := r.db.Where(" username=?", _username).First(&jason).Error

	return jason, err
}
func (r *UserRepository) GetByChatID(chatID int64) (domain.User, error) {
	user := domain.User{}
	err := r.db.Where(" associates=?", chatID).First(&user).Error
	return user, err
}

func (r *UserRepository) ListRegisteredUsers(ctx context.Context, botName string) ([]domain.User, error) {
	users := make([]domain.User, 0)

	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("associates IS NOT NULL").
		Where("associates <> ''").
		Order("created_at ASC")

	if botName != "" {
		query = query.Where("bot_name = ?", botName)
	}

	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) GetByAssociates(associates string) (domain.User, error) {
	user := domain.User{}
	err := r.db.Where(" associates=?", associates).First(&user).Error
	return user, err
}
func (r *UserRepository) UpdateLang(_lang string, _chatID int64) error {
	query := "UPDATE tg_users SET lang = ? WHERE associates = ?"
	tx := r.db.Exec(query, _lang, _chatID)
	return tx.Error
}

func (r *UserRepository) FetchNewestAddress() ([]domain.User, error) {
	query := `SELECT address,associates
    FROM 
      sys_address  where disable=0 ;
    `
	var addresses []domain.User
	r.db.Select(&addresses, query)
	return addresses, nil
}
func (r *UserRepository) DisableTronAddress(_address string) error {
	query := "UPDATE sys_address SET disable = 1 WHERE address = ?"
	tx := r.db.Exec(query, _address)
	return tx.Error
}

func (r *UserRepository) BindChat(_associates string, _username string) error {
	query := "UPDATE tg_users SET associates = ? WHERE username = ?"
	tx := r.db.Exec(query, _associates, _username)
	return tx.Error
}

func (r *UserRepository) BindTronAddress(_address string, _username string) error {
	query := "UPDATE tg_users SET tron_address = ? WHERE username = ?"
	tx := r.db.Exec(query, _address, _username)
	return tx.Error
}

func (r *UserRepository) BindEthereumAddress(_address string, _username string) error {
	query := "UPDATE tg_users SET eth_address = ? WHERE username = ?"
	tx := r.db.Exec(query, _address, _username)
	return tx.Error
}

func (r *UserRepository) NotifyTronAddress() ([]domain.User, error) {
	query := `SELECT t.username,t.tron_address,t.associates
    FROM
        tg_users t
    LEFT JOIN
        sys_address s ON t.tron_address = s.address

    WHERE s.disable = 0
    `
	var addresses []domain.User
	r.db.Select(&addresses, query)
	return addresses, nil
}
func (r *UserRepository) NotifyEthereumAddress() ([]domain.User, error) {
	query := `SELECT t.username,t.eth_address,t.associates
    FROM
        tg_users t
    LEFT JOIN
        sys_address s ON t.eth_address = s.address

    WHERE s.disable = 0
    `
	var addresses []domain.User
	r.db.Select(&addresses, query)
	return addresses, nil
}
