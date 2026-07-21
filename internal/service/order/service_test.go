package order

import (
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"ushield_bot/internal/cache"
	"ushield_bot/internal/config"
	"ushield_bot/internal/testsupport"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestHandleBalancePaymentInsufficientBalance(t *testing.T) {
	testsupport.SeedTranslations()

	db, mock, cleanupDB := testsupport.NewMockGormDB(t)
	defer cleanupDB()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	mock.ExpectQuery("SELECT \\* FROM `user_usdt_deposits` WHERE order_no = \\?").
		WithArgs("ORDER001").
		WillReturnRows(testsupport.NewRows([]string{"id", "user_id", "status", "placeholder", "order_no", "amount"}).
			AddRow(1, 10001, 0, "0.001", "ORDER001", "10"))

	mock.ExpectQuery("SELECT \\* FROM `tg_users` WHERE .*associates=\\?.*LIMIT \\?").
		WithArgs(10001, 1).
		WillReturnRows(testsupport.NewRows([]string{"id", "associates", "username", "amount"}).
			AddRow(7, "10001", "alice", "3"))

	service := NewService(&config.Config{}, db, bot, cache.NewMemoryCache())
	service.HandleBalancePayment(newCallbackQuery(10001), "ORDER001")

	requests := recorder.Requests()
	require.Len(t, requests, 1)
	require.Equal(t, "sendMessage", requests[0].Method)
	require.Contains(t, requests[0].Form.Get("text"), "余额不足")
	require.Contains(t, requests[0].Form.Get("text"), "@alice")
}

func TestHandleBalancePaymentSuccess(t *testing.T) {
	testsupport.SeedTranslations()

	db, mock, cleanupDB := testsupport.NewMockGormDB(t)
	defer cleanupDB()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	mock.ExpectQuery("SELECT \\* FROM `user_usdt_deposits` WHERE order_no = \\?").
		WithArgs("ORDER002").
		WillReturnRows(testsupport.NewRows([]string{"id", "user_id", "status", "placeholder", "order_no", "amount"}).
			AddRow(2, 10002, 0, "0.002", "ORDER002", "5"))

	mock.ExpectQuery("SELECT \\* FROM `tg_users` WHERE .*associates=\\?.*LIMIT \\?").
		WithArgs(10002, 1).
		WillReturnRows(testsupport.NewRows([]string{"id", "associates", "username", "amount"}).
			AddRow(9, "10002", "bob", "12"))

	mock.ExpectExec("UPDATE `tg_users` SET .* WHERE `id` = \\?").
		WillReturnResult(testsupport.SQLResult(0, 1))
	mock.ExpectExec("UPDATE `user_usdt_placeholders` SET `status`=\\? WHERE placeholder = \\?").
		WithArgs(0, "0.002").
		WillReturnResult(testsupport.SQLResult(0, 1))
	mock.ExpectExec("UPDATE `user_usdt_deposits` SET `status`=\\?,`updated_at`=\\? WHERE order_no = \\?").
		WithArgs(2, sqlmock.AnyArg(), "ORDER002").
		WillReturnResult(testsupport.SQLResult(0, 1))

	service := NewService(&config.Config{}, db, bot, cache.NewMemoryCache())
	service.HandleBalancePayment(newCallbackQuery(10002), "ORDER002")

	requests := recorder.Requests()
	require.Len(t, requests, 1)
	require.Equal(t, "sendMessage", requests[0].Method)
	require.Contains(t, requests[0].Form.Get("text"), "TOPUP-ORDER002")
	require.Contains(t, requests[0].Form.Get("text"), "成功购买价值5USDT")
}

func TestCreateTopupOrderFallsBackToAgentEnv(t *testing.T) {
	testsupport.SeedTranslations()

	oldAgent, existed := os.LookupEnv("AGENT")
	require.NoError(t, os.Setenv("AGENT", "admin"))
	defer func() {
		if existed {
			_ = os.Setenv("AGENT", oldAgent)
			return
		}
		_ = os.Unsetenv("AGENT")
	}()

	imageFile, err := os.CreateTemp(t.TempDir(), "order-*.png")
	require.NoError(t, err)
	require.NoError(t, imageFile.Close())

	db, mock, cleanupDB := testsupport.NewMockGormDB(t)
	defer cleanupDB()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	cacheStore := cache.NewMemoryCache()
	require.NoError(t, cacheStore.Set("LANG_10003", "zh", 0))

	mock.ExpectQuery("SELECT \\* FROM `polytopup_topup_plan` WHERE .*id =\\? .*LIMIT \\?").
		WithArgs("1", 1).
		WillReturnRows(testsupport.NewRows([]string{"id", "name_cn", "price"}).
			AddRow(1, "10元话费", "10"))
	mock.ExpectQuery("SELECT \\* FROM `user_usdt_placeholders` WHERE status = \\? ORDER BY RAND\\(\\)").
		WithArgs(0).
		WillReturnRows(testsupport.NewRows([]string{"id", "status", "placeholder"}).
			AddRow(1, 0, "0.001"))
	mock.ExpectExec("UPDATE `user_usdt_placeholders` SET `status`=\\? WHERE id = \\?").
		WithArgs(1, 1).
		WillReturnResult(testsupport.SQLResult(0, 1))
	mock.ExpectQuery("SELECT address, deposit_address FROM `sys_users` WHERE username = \\? ORDER BY `sys_users`.`username` LIMIT \\?").
		WithArgs("admin", 1).
		WillReturnRows(testsupport.NewRows([]string{"address", "deposit_address"}).
			AddRow("TMAIN", "TDEPOSIT123"))
	mock.ExpectExec("INSERT INTO `user_usdt_deposits`").
		WillReturnResult(testsupport.SQLResult(11, 1))

	service := NewService(&config.Config{OrderImagePath: imageFile.Name()}, db, bot, cacheStore)
	service.CreateTopupOrder(10003, "alice", "+86123456789", "1", ProductTopup)

	requests := recorder.Requests()
	require.Len(t, requests, 1)
	require.Equal(t, "sendPhoto", requests[0].Method)
	require.Contains(t, requests[0].Form.Get("caption"), "TDEPOSIT123")
	require.Contains(t, requests[0].Form.Get("caption"), "点击地址复制")
}

func TestCreateTopupOrderNotifiesSupportWhenDepositAddressMissing(t *testing.T) {
	testsupport.SeedTranslations()

	oldAgent, existed := os.LookupEnv("AGENT")
	require.NoError(t, os.Setenv("AGENT", "masion"))
	defer func() {
		if existed {
			_ = os.Setenv("AGENT", oldAgent)
			return
		}
		_ = os.Unsetenv("AGENT")
	}()

	db, mock, cleanupDB := testsupport.NewMockGormDB(t)
	defer cleanupDB()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	cacheStore := cache.NewMemoryCache()
	require.NoError(t, cacheStore.Set("LANG_10004", "zh", 0))

	mock.ExpectQuery("SELECT \\* FROM `polytopup_topup_plan` WHERE .*id =\\? .*LIMIT \\?").
		WithArgs("1", 1).
		WillReturnRows(testsupport.NewRows([]string{"id", "name_cn", "price"}).
			AddRow(1, "10元话费", "10"))
	mock.ExpectQuery("SELECT \\* FROM `user_usdt_placeholders` WHERE status = \\? ORDER BY RAND\\(\\)").
		WithArgs(0).
		WillReturnRows(testsupport.NewRows([]string{"id", "status", "placeholder"}).
			AddRow(1, 0, "0.001"))
	mock.ExpectExec("UPDATE `user_usdt_placeholders` SET `status`=\\? WHERE id = \\?").
		WithArgs(1, 1).
		WillReturnResult(testsupport.SQLResult(0, 1))
	mock.ExpectQuery("SELECT address, deposit_address FROM `sys_users` WHERE username = \\? ORDER BY `sys_users`.`username` LIMIT \\?").
		WithArgs("masion", 1).
		WillReturnRows(testsupport.NewRows([]string{"address", "deposit_address"}).
			AddRow("TMAIN", ""))

	service := NewService(&config.Config{NotifyChatID: 99999}, db, bot, cacheStore)
	service.CreateTopupOrder(10004, "alice", "+86123456789", "1", ProductTopup)

	requests := recorder.Requests()
	require.Len(t, requests, 1)
	require.Equal(t, "sendMessage", requests[0].Method)
	require.Equal(t, "99999", requests[0].Form.Get("chat_id"))
	require.Contains(t, requests[0].Form.Get("text"), "下单异常告警")
	require.Contains(t, requests[0].Form.Get("text"), "用户 masion 的收款地址为空")
	require.Contains(t, requests[0].Form.Get("text"), "@alice")
}

func newCallbackQuery(chatID int64) *tgbotapi.CallbackQuery {
	return &tgbotapi.CallbackQuery{
		ID:   "callback-id",
		Data: "test-data",
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{
				ID: chatID,
			},
		},
	}
}
