package telegram

import (
	"strconv"

	"ushield_bot/internal/cache"
	orderservice "ushield_bot/internal/service/order"
	profileservice "ushield_bot/internal/service/profile"
	topupservice "ushield_bot/internal/service/topup"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type callbackDispatcher struct {
	cache          cache.Cache
	profileService profileservice.Handler
	topupService   topupservice.Handler
	orderService   orderservice.Handler
}

func NewCallbackDispatcher(
	cache cache.Cache,
	profileService profileservice.Handler,
	topupService topupservice.Handler,
	orderService orderservice.Handler,
) CallbackDispatcher {
	return &callbackDispatcher{
		cache:          cache,
		profileService: profileService,
		topupService:   topupService,
		orderService:   orderService,
	}
}

func (d *callbackDispatcher) Handle(callbackQuery *tgbotapi.CallbackQuery) {
	data := callbackQuery.Data

	switch {
	case hasPrefix(data, "nope_purchase_order"):
		d.orderService.HandleBalancePayment(callbackQuery, trimPrefix(data, "nope_purchase_order"))
	case data == callbackCancelOrder:
		d.orderService.DeleteCachedOrderMessage(callbackQuery.Message.Chat.ID)
	case hasPrefix(data, "click_country_"):
		d.topupService.ShowPlanMenu(callbackQuery.Message.Chat.ID, trimPrefix(data, "click_country_"), orderservice.ProductTopup)
	case hasPrefix(data, "click_data_"):
		d.topupService.ShowPlanMenu(callbackQuery.Message.Chat.ID, trimPrefix(data, "click_data_"), orderservice.ProductData)
	case hasPrefix(data, "click_plan_"):
		countryID, planID, ok := splitPair(trimPrefix(data, "click_plan_"))
		if ok {
			d.topupService.ShowPlanMobilePrompt(callbackQuery.Message.Chat.ID, countryID, planID, orderservice.ProductTopup)
		}
	case hasPrefix(data, "click_5G_data_"):
		countryID, planID, ok := splitPair(trimPrefix(data, "click_5G_data_"))
		if ok {
			d.topupService.ShowPlanMobilePrompt(callbackQuery.Message.Chat.ID, countryID, planID, orderservice.ProductData)
		}
	case hasPrefix(data, "click_mobile_topup"):
		mobile, planID, ok := splitPair(trimPrefix(data, "click_mobile_topup"))
		if ok {
			d.orderService.CreateTopupOrder(callbackQuery.Message.Chat.ID, callbackQuery.Message.Chat.UserName, mobile, planID, orderservice.ProductTopup)
		}
	case hasPrefix(data, "click_mobile_data_topup"):
		mobile, planID, ok := splitPair(trimPrefix(data, "click_mobile_data_topup"))
		if ok {
			d.orderService.CreateTopupOrder(callbackQuery.Message.Chat.ID, callbackQuery.Message.Chat.UserName, mobile, planID, orderservice.ProductData)
		}
	case data == callbackBackTopup:
		d.orderService.DeleteCachedOrderMessage(callbackQuery.Message.Chat.ID)
		d.topupService.ShowCountryMenu(callbackQuery.Message.Chat.ID, orderservice.ProductTopup)
	case data == callbackBackData:
		d.orderService.DeleteCachedOrderMessage(callbackQuery.Message.Chat.ID)
		d.topupService.ShowCountryMenu(callbackQuery.Message.Chat.ID, orderservice.ProductData)
	case hasPrefix(data, "click_mobile_mgr"):
		d.topupService.ShowMobileManager(callbackQuery.Message.Chat.ID, trimPrefix(data, "click_mobile_mgr"), false)
	case hasPrefix(data, topupservice.ActionAddMobile):
		d.topupService.PromptMobileManagerInput(callbackQuery.Message.Chat.ID, trimPrefix(data, topupservice.ActionAddMobile), topupservice.ActionAddMobile)
	case hasPrefix(data, topupservice.ActionDeleteMobile):
		d.topupService.PromptMobileManagerInput(callbackQuery.Message.Chat.ID, trimPrefix(data, topupservice.ActionDeleteMobile), topupservice.ActionDeleteMobile)
	case data == "deposit_amount":
		d.profileService.ShowDepositAmountOptions("zh", callbackQuery)
	case data == callbackBackProfile:
		d.profileService.ShowHome("zh", callbackQuery.Message)
	case hasPrefix(data, "deposit_usdt"):
		d.orderService.ShowDepositUSDTOrder("zh", callbackQuery)
	case data == "click_deposit_usdt_records":
		d.profileService.ShowDepositUSDTRecords("zh", callbackQuery)
	case data == "prev_deposit_usdt_page":
		if _, done := d.profileService.ShowPrevDepositUSDTPage("zh", callbackQuery); done {
			return
		}
	case data == "next_deposit_usdt_page":
		if d.profileService.ShowNextDepositUSDTPage("zh", callbackQuery) {
			return
		}
	default:
		status, _ := d.cache.Get(strconv.FormatInt(callbackQuery.Message.Chat.ID, 10))
		logrus.WithFields(logrus.Fields{
			"chat_id": callbackQuery.Message.Chat.ID,
			"data":    data,
			"status":  status,
		}).Info("未匹配的回调")
	}
}
