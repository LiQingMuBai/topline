package telegram

import (
	"strconv"
	"strings"

	"ushield_bot/internal/cache"
	orderservice "ushield_bot/internal/service/order"
	profileservice "ushield_bot/internal/service/profile"
	topupservice "ushield_bot/internal/service/topup"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type messageDispatcher struct {
	cache          cache.Cache
	langResolver   *userLanguageResolver
	profileService profileservice.Handler
	topupService   topupservice.Handler
}

func NewMessageDispatcher(
	cache cache.Cache,
	langResolver *userLanguageResolver,
	profileService profileservice.Handler,
	topupService topupservice.Handler,
) MessageDispatcher {
	return &messageDispatcher{
		cache:          cache,
		langResolver:   langResolver,
		profileService: profileService,
		topupService:   topupService,
	}
}

func (d *messageDispatcher) Handle(message *tgbotapi.Message) {
	if message.IsCommand() {
		d.handleCommand(message)
		return
	}
	d.handleRegularMessage(message)
}

func (d *messageDispatcher) handleCommand(message *tgbotapi.Message) {
	switch {
	case message.Command() == "start":
		if err := d.profileService.EnsureUser(message); err != nil {
			logrus.WithError(err).Error("初始化用户失败")
			return
		}
		d.profileService.ShowStartKeyboard(message)
	case message.Command() == "hide":
		d.profileService.HideKeyboard(message)
	case hasPrefix(message.Command(), topupservice.StateSetReminderPrefix):
		d.topupService.PromptReminderDay(message.Chat.ID, trimPrefix(message.Command(), topupservice.StateSetReminderPrefix))
	}
}

func (d *messageDispatcher) handleRegularMessage(message *tgbotapi.Message) {
	switch message.Text {
	case menuSupport:
		d.profileService.ShowSupport(message.Chat.ID)
	case menuData:
		d.topupService.ShowCountryMenu(message.Chat.ID, orderservice.ProductData)
	case menuTopup:
		d.topupService.ShowCountryMenu(message.Chat.ID, orderservice.ProductTopup)
	case menuProfile:
		d.profileService.ShowHome(d.langResolver.Resolve(message.Chat.ID), message)
	default:
		d.handlePendingMessage(message)
	}
}

func (d *messageDispatcher) handlePendingMessage(message *tgbotapi.Message) {
	status, _ := d.cache.Get(strconv.FormatInt(message.Chat.ID, 10))
	switch {
	case hasPrefix(status, topupservice.StateSetReminderPrefix):
		d.topupService.HandleReminderDayInput(message.Chat.ID, message.Text, trimPrefix(status, topupservice.StateSetReminderPrefix))
	case hasPrefix(status, topupservice.StateTopupMobilePrefix):
		d.topupService.HandleMobileInput(message.Chat.ID, message.Chat.UserName, message.Text, trimPrefix(status, topupservice.StateTopupMobilePrefix), orderservice.ProductTopup)
	case hasPrefix(status, topupservice.StateDataMobilePrefix):
		d.topupService.HandleMobileInput(message.Chat.ID, message.Chat.UserName, message.Text, trimPrefix(status, topupservice.StateDataMobilePrefix), orderservice.ProductData)
	case hasPrefix(status, topupservice.ActionAddMobile):
		d.topupService.HandleAddMobileInput(message.Chat.ID, strings.TrimSpace(message.Text), trimPrefix(status, topupservice.ActionAddMobile))
	case hasPrefix(status, topupservice.ActionDeleteMobile):
		d.topupService.HandleDeleteMobileInput(message.Chat.ID, strings.TrimSpace(message.Text), trimPrefix(status, topupservice.ActionDeleteMobile))
	}
}
