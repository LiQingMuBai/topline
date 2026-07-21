package profile

import (
	"testing"

	"github.com/stretchr/testify/require"
	"ushield_bot/internal/cache"
	"ushield_bot/internal/config"
	"ushield_bot/internal/global"
	"ushield_bot/internal/testsupport"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestShowPrevDepositUSDTPageAtFirstPageReturnsNoop(t *testing.T) {
	testsupport.SeedTranslations()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	global.DepositStates = map[int64]*global.DepositState{
		10001: {CurrentPage: 1},
	}

	service := NewService(&config.Config{}, nil, bot, cache.NewMemoryCache())
	state, stop := service.ShowPrevDepositUSDTPage("zh", newProfileCallbackQuery(10001))

	require.True(t, stop)
	require.Nil(t, state)
	require.Empty(t, recorder.Requests())
}

func TestShowPrevDepositUSDTPageRendersPreviousPage(t *testing.T) {
	testsupport.SeedTranslations()

	db, mock, cleanupDB := testsupport.NewMockGormDB(t)
	defer cleanupDB()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	global.DepositStates = map[int64]*global.DepositState{
		10002: {CurrentPage: 2},
	}

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `user_usdt_deposits` WHERE user_id = \\? AND status = \\?").
		WithArgs(10002, 1).
		WillReturnRows(testsupport.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT id,amount,order_no, DATE_FORMAT\\(created_at, '%m-%d'\\) as created_date FROM `user_usdt_deposits` WHERE user_id = \\? AND status = \\? ORDER BY id DESC LIMIT \\?").
		WithArgs(10002, 1, 10).
		WillReturnRows(testsupport.NewRows([]string{"id", "amount", "order_no", "created_date"}).
			AddRow(1, "25", "ORDER888", "07-21"))

	service := NewService(&config.Config{}, db, bot, cache.NewMemoryCache())
	state, stop := service.ShowPrevDepositUSDTPage("zh", newProfileCallbackQuery(10002))

	require.False(t, stop)
	require.NotNil(t, state)
	require.EqualValues(t, 1, state.CurrentPage)

	requests := recorder.Requests()
	require.Len(t, requests, 1)
	require.Equal(t, "sendMessage", requests[0].Method)
	require.Contains(t, requests[0].Form.Get("text"), "ORDER888")
	require.Contains(t, requests[0].Form.Get("text"), "25 USDT")
}

func newProfileCallbackQuery(chatID int64) *tgbotapi.CallbackQuery {
	return &tgbotapi.CallbackQuery{
		ID:   "profile-callback",
		Data: "pagination",
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{
				ID: chatID,
			},
		},
	}
}
