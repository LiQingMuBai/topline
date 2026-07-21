package order

import (
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
