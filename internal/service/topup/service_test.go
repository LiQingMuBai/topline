package topup

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"ushield_bot/internal/cache"
	"ushield_bot/internal/config"
	"ushield_bot/internal/poly/model"
	orderservice "ushield_bot/internal/service/order"
	"ushield_bot/internal/testsupport"
)

func TestShowPlanMobilePromptWithoutSavedMobileSetsInputState(t *testing.T) {
	testsupport.SeedTranslations()

	db, mock, cleanupDB := testsupport.NewMockGormDB(t)
	defer cleanupDB()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	memCache := cache.NewMemoryCache()

	mock.ExpectQuery("SELECT \\* FROM `polytopup_country` WHERE .*id =\\?.*deleted_at.*LIMIT \\?").
		WithArgs("86", 1).
		WillReturnRows(testsupport.NewRows([]string{"id", "name_cn", "name_en"}).
			AddRow(86, "中国", "China"))
	mock.ExpectQuery("SELECT `id`,mobile,reminder_day FROM `polytopup_user_mobile` WHERE .*country_id = \\? and chat_id = \\?.*deleted_at.*").
		WithArgs("86", "10001").
		WillReturnRows(testsupport.NewRows([]string{"id", "mobile", "reminder_day"}))

	service := NewService(&config.Config{}, db, bot, memCache, nil)
	service.ShowPlanMobilePrompt(10001, "86", "P100", "topup")

	state, err := memCache.Get("10001")
	require.NoError(t, err)
	require.Equal(t, StateTopupMobilePrefix+"86=P100", state)

	requests := recorder.Requests()
	require.Len(t, requests, 1)
	require.Equal(t, "sendMessage", requests[0].Method)
	require.Contains(t, requests[0].Form.Get("text"), "请输入您要充值的")
	require.Contains(t, requests[0].Form.Get("text"), "中国")
	require.Contains(t, requests[0].Form.Get("text"), "移动 +86138123456789")
}

func TestHandleReminderDayInputUpdatesReminderAndRendersManager(t *testing.T) {
	testsupport.SeedTranslations()

	db, mock, cleanupDB := testsupport.NewMockGormDB(t)
	defer cleanupDB()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	mock.ExpectExec("UPDATE `polytopup_user_mobile` SET `reminder_day`=\\?,`updated_at`=\\? WHERE id = \\? AND `polytopup_user_mobile`.`deleted_at` IS NULL").
		WithArgs(18, sqlmock.AnyArg(), "9").
		WillReturnResult(testsupport.SQLResult(0, 1))
	mock.ExpectQuery("SELECT `id`,mobile,reminder_day,chat_id,country_id FROM `polytopup_user_mobile` WHERE .*id = \\?.*deleted_at.*").
		WithArgs("9").
		WillReturnRows(testsupport.NewRows([]string{"id", "mobile", "reminder_day", "chat_id", "country_id"}).
			AddRow(9, "+8613800000000", "18", 10002, 86))
	mock.ExpectQuery("SELECT \\* FROM `polytopup_country` WHERE .*id =\\?.*deleted_at.*LIMIT \\?").
		WithArgs("86", 1).
		WillReturnRows(testsupport.NewRows([]string{"id", "name_cn", "name_en"}).
			AddRow(86, "中国", "China"))
	mock.ExpectQuery("SELECT `id`,mobile,reminder_day FROM `polytopup_user_mobile` WHERE .*country_id = \\? and chat_id = \\?.*deleted_at.*").
		WithArgs("86", "10002").
		WillReturnRows(testsupport.NewRows([]string{"id", "mobile", "reminder_day"}).
			AddRow(9, "+8613800000000", "18"))

	service := NewService(&config.Config{}, db, bot, cache.NewMemoryCache(), nil)
	service.HandleReminderDayInput(10002, "18", "9")

	requests := recorder.Requests()
	require.Len(t, requests, 2)
	require.Equal(t, "sendMessage", requests[0].Method)
	require.Contains(t, requests[0].Form.Get("text"), "提醒日，设置成功")
	require.Equal(t, "sendMessage", requests[1].Method)
	require.Contains(t, requests[1].Form.Get("text"), "中国")
	require.Contains(t, requests[1].Form.Get("text"), "提醒18号")
}

func TestHandleAddMobileInputRejectsEntryWithoutCarrierName(t *testing.T) {
	testsupport.SeedTranslations()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	service := NewService(&config.Config{}, nil, bot, cache.NewMemoryCache(), nil)
	service.HandleAddMobileInput(10003, "+86138123456789", "86")

	requests := recorder.Requests()
	require.Len(t, requests, 1)
	require.Equal(t, "sendMessage", requests[0].Method)
	require.Contains(t, requests[0].Form.Get("text"), "运营商 + 手机号码")
	require.Contains(t, requests[0].Form.Get("text"), "移动 +86138123456789")
}

func TestValidMobileEntryAcceptsEntryWithoutSpace(t *testing.T) {
	require.True(t, validMobileEntry("移动 +86138123456789"))
	require.True(t, validMobileEntry("移动+86138123456789"))
	require.False(t, validMobileEntry("+86138123456789"))
}

func TestShowCountryMenuUsesConfiguredWorkTime(t *testing.T) {
	testsupport.SeedTranslations()

	db, mock, cleanupDB := testsupport.NewMockGormDB(t)
	defer cleanupDB()

	bot, recorder, cleanupBot := testsupport.NewTestBot(t)
	defer cleanupBot()

	mock.ExpectQuery("SELECT \\* FROM `polytopup_country` WHERE `polytopup_country`.`deleted_at` IS NULL").
		WillReturnRows(testsupport.NewRows([]string{"id", "name_cn"}).
			AddRow(86, "中国"))

	service := NewService(&config.Config{SupportWorkTime: "10:00-20:00"}, db, bot, cache.NewMemoryCache(), nil)
	service.ShowCountryMenu(10004, "topup")

	requests := recorder.Requests()
	require.Len(t, requests, 1)
	require.Equal(t, "sendMessage", requests[0].Method)
	require.Contains(t, requests[0].Form.Get("text"), "10:00-20:00")
}

func TestPlanPromptUsesConfiguredFeeRate(t *testing.T) {
	testsupport.SeedTranslations()

	memCache := cache.NewMemoryCache()
	require.NoError(t, memCache.Set("LANG_10005", "en", 0))

	service := NewService(&config.Config{TopupFeeRate: "6%-9%"}, nil, nil, nil, nil)

	service.cache = memCache
	text := service.planPrompt(10005, orderservice.ProductTopup)

	require.Contains(t, text, "6%-9%")
	require.Contains(t, text, "service fee")
}

func TestBuildCountryKeyboardRowsGroupsByContinent(t *testing.T) {
	testsupport.SeedTranslations()

	rows := buildCountryKeyboardRows("en", orderservice.ProductTopup, []model.Country{
		{BaseModel: model.BaseModel{ID: 1}, NameCn: "中国", NameEn: "China"},
		{BaseModel: model.BaseModel{ID: 2}, NameCn: "德国", NameEn: "Germany"},
		{BaseModel: model.BaseModel{ID: 3}, NameCn: "肯尼亚", NameEn: "Kenya"},
		{BaseModel: model.BaseModel{ID: 4}, NameCn: "加拿大", NameEn: "Canada"},
		{BaseModel: model.BaseModel{ID: 5}, NameCn: "澳大利亚", NameEn: "Australia"},
	})

	require.Len(t, rows, 10)
	require.Equal(t, "🌏 Asia", rows[0][0].Text)
	require.Equal(t, "China", rows[1][0].Text)
	require.Equal(t, "🌍 Europe", rows[2][0].Text)
	require.Equal(t, "Germany", rows[3][0].Text)
	require.Equal(t, "🌍 Africa", rows[4][0].Text)
	require.Equal(t, "Kenya", rows[5][0].Text)
	require.Equal(t, "🌎 Americas", rows[6][0].Text)
	require.Equal(t, "Canada", rows[7][0].Text)
	require.Equal(t, "🌏 Oceania", rows[8][0].Text)
	require.Equal(t, "Australia", rows[9][0].Text)
	require.NotNil(t, rows[1][0].CallbackData)
	require.Equal(t, "click_country_1", *rows[1][0].CallbackData)
}
