package telegram

import (
	"ushield_bot/internal/cache"
	"ushield_bot/internal/config"
	orderservice "ushield_bot/internal/service/order"
	profileservice "ushield_bot/internal/service/profile"
	topupservice "ushield_bot/internal/service/topup"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

// Router 管理 Telegram 更新分发，避免 main.go 堆积业务逻辑。
type Router struct {
	messageDispatcher  MessageDispatcher
	callbackDispatcher CallbackDispatcher
}

// NewRouter 构造 Telegram 路由器。
func NewRouter(cfg *config.Config, db *gorm.DB, bot *tgbotapi.BotAPI, cache cache.Cache) *Router {
	orderSvc := orderservice.NewService(cfg, db, bot, cache)
	profileSvc := profileservice.NewService(cfg, db, bot, cache)
	topupSvc := topupservice.NewService(db, bot, cache, orderSvc)
	langResolver := newUserLanguageResolver(db, cache)

	return &Router{
		messageDispatcher:  NewMessageDispatcher(cache, langResolver, profileSvc, topupSvc),
		callbackDispatcher: NewCallbackDispatcher(cache, profileSvc, topupSvc, orderSvc),
	}
}

// HandleUpdate 根据更新类型分发到对应处理器。
func (r *Router) HandleUpdate(update tgbotapi.Update) {
	switch {
	case update.Message != nil:
		r.messageDispatcher.Handle(update.Message)
	case update.CallbackQuery != nil:
		r.callbackDispatcher.Handle(update.CallbackQuery)
	}
}
