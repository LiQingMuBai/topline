package telegram

import (
	"strconv"
	"time"

	"ushield_bot/internal/cache"
	"ushield_bot/internal/global"
	"ushield_bot/internal/i18n"
	"ushield_bot/internal/infrastructure/repositories"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

const (
	langCachePrefix = "LANG_"
	langTTL         = 24 * time.Hour

	callbackCancelOrder = "cancel_order"
	callbackBackTopup   = "back_home"
	callbackBackData    = "back_home_data"
	callbackBackProfile = "back_profile"
	callbackSetLang     = "set_lang:"
)

type userLanguageResolver struct {
	db    *gorm.DB
	cache cache.Cache
}

func newUserLanguageResolver(db *gorm.DB, cache cache.Cache) *userLanguageResolver {
	return &userLanguageResolver{
		db:    db,
		cache: cache,
	}
}

func (r *userLanguageResolver) Resolve(chatID int64) string {
	lang, err := r.cache.Get(r.langKey(chatID))
	if err == nil && lang != "" {
		return lang
	}

	userRepo := repositories.NewUserRepository(r.db)
	record, repoErr := userRepo.GetByChatID(chatID)
	if repoErr == nil && record.Lang != "" {
		_ = r.cache.Set(r.langKey(chatID), record.Lang, langTTL)
		return record.Lang
	}
	return global.DefaultLang
}

func (r *userLanguageResolver) langKey(chatID int64) string {
	return langCachePrefix + strconv.FormatInt(chatID, 10)
}

func hasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}

func trimPrefix(value, prefix string) string {
	if hasPrefix(value, prefix) {
		return value[len(prefix):]
	}
	return value
}

func splitPair(value string) (string, string, bool) {
	for i := 0; i < len(value); i++ {
		if value[i] == '=' {
			return value[:i], value[i+1:], true
		}
	}
	return "", "", false
}

func matchesMenuText(text, key string) bool {
	for _, lang := range global.SupportedLangs {
		if text == i18n.T(lang, key) {
			return true
		}
	}
	return false
}

type MessageDispatcher interface {
	Handle(message *tgbotapi.Message)
}

type CallbackDispatcher interface {
	Handle(callbackQuery *tgbotapi.CallbackQuery)
}
