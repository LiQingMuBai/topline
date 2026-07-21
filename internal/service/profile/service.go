package profile

import (
	"context"
	"strconv"
	"strings"
	"time"

	"ushield_bot/internal/cache"
	"ushield_bot/internal/config"
	"ushield_bot/internal/domain"
	"ushield_bot/internal/global"
	"ushield_bot/internal/infrastructure/repositories"
	toolkit "ushield_bot/internal/infrastructure/tools"
	"ushield_bot/internal/request"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Handler 抽象个人中心相关能力，便于测试替换与依赖解耦。
type Handler interface {
	EnsureUser(message *tgbotapi.Message) error
	ShowStartKeyboard(message *tgbotapi.Message)
	HideKeyboard(message *tgbotapi.Message)
	ShowHome(lang string, message *tgbotapi.Message)
	ShowSupport(chatID int64)
	ShowDepositAmountOptions(lang string, callbackQuery *tgbotapi.CallbackQuery)
	ShowDepositUSDTRecords(lang string, callbackQuery *tgbotapi.CallbackQuery)
	ShowPrevDepositUSDTPage(lang string, callbackQuery *tgbotapi.CallbackQuery) (*global.DepositState, bool)
	ShowNextDepositUSDTPage(lang string, callbackQuery *tgbotapi.CallbackQuery) bool
}

// Service 负责个人中心与客服信息相关能力。
type Service struct {
	cfg   *config.Config
	db    *gorm.DB
	bot   *tgbotapi.BotAPI
	cache cache.Cache
}

// NewService 创建个人中心服务。
func NewService(cfg *config.Config, db *gorm.DB, bot *tgbotapi.BotAPI, cache cache.Cache) *Service {
	return &Service{
		cfg:   cfg,
		db:    db,
		bot:   bot,
		cache: cache,
	}
}

// EnsureUser 确保当前 Telegram 用户已经完成初始化。
func (s *Service) EnsureUser(message *tgbotapi.Message) error {
	userRepo := repositories.NewUserRepository(s.db)
	parentUID := s.resolveParentUID(message.Text, userRepo)

	record, err := userRepo.GetByChatID(message.Chat.ID)
	if err != nil {
		user := domain.User{
			Associates: strconv.FormatInt(message.Chat.ID, 10),
			Username:   message.Chat.UserName,
			Lang:       "zh",
			CreatedAt:  time.Now(),
			BotName:    s.cfg.BotName,
		}
		if parentUID != "" {
			user.ParentUserID = parentUID
		}
		if err := userRepo.Create(context.Background(), &user); err != nil {
			return err
		}
		record = user
	}

	if err := userRepo.UpdateUsernameByChatID(message.From.UserName, message.Chat.ID); err != nil {
		logrus.WithError(err).Warn("更新用户名失败")
	}

	lang := record.Lang
	if lang == "" {
		lang = "zh"
	}
	if err := s.cache.Set("LANG_"+strconv.FormatInt(message.Chat.ID, 10), lang, 24*time.Hour); err != nil {
		logrus.WithError(err).Warn("写入语言缓存失败")
	}

	return nil
}

// ShowStartKeyboard 展示主菜单键盘。
func (s *Service) ShowStartKeyboard(message *tgbotapi.Message) {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⛽话费充值"),
			tgbotapi.NewKeyboardButton("🔋流量充值"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("👤个人中心"),
			tgbotapi.NewKeyboardButton("🐍联系娘子"),
		),
	)
	keyboard.OneTimeKeyboard = false
	keyboard.ResizeKeyboard = true

	msg := tgbotapi.NewMessage(message.Chat.ID, global.Translations["zh"]["welcome_tips"])
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = keyboard
	_, _ = s.bot.Send(msg)
}

// HideKeyboard 隐藏主菜单键盘。
func (s *Service) HideKeyboard(message *tgbotapi.Message) {
	msg := tgbotapi.NewMessage(message.Chat.ID, "键盘已隐藏，发送 /start 重新显示")
	msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
	_, _ = s.bot.Send(msg)
}

// ShowHome 展示个人中心首页。
func (s *Service) ShowHome(lang string, message *tgbotapi.Message) {
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳"+global.Translations[lang]["deposit"], "deposit_amount"),
			tgbotapi.NewInlineKeyboardButtonData("📄"+global.Translations[lang]["billing"], "click_deposit_usdt_records"),
		),
	)

	userRepo := repositories.NewUserRepository(s.db)
	user, _ := userRepo.GetByChatID(message.Chat.ID)
	if toolkit.IsEmpty(user.Amount) {
		user.Amount = "0"
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, "🆔 "+global.Translations[lang]["user_id"]+"："+user.Associates+"\n\n👤 "+global.Translations[lang]["username"]+"：@"+user.Username+"\n\n"+
		global.Translations[lang]["balance"]+"：\n"+
		"- USDT："+user.Amount+"\n"+
		"- "+global.Translations[lang]["promotion_income"]+"："+user.PromotionIncome+" USDT"+"\n\n"+
		global.Translations[lang]["promotion_link"]+":"+"<code>"+"https://t.me/PolyTopUp_bot?start="+strconv.FormatInt(message.Chat.ID, 10)+"</code>",
	)
	msg.ReplyMarkup = inlineKeyboard
	msg.ParseMode = "HTML"
	s.send(msg)
}

// ShowSupport 展示客服联系方式。
func (s *Service) ShowSupport(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "📞"+global.Translations["zh"]["support"]+"：@PolyTopUp\n工作时间：9:00-22:00\n")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙"+global.Translations["zh"]["back_home"], "back_profile"),
		),
	)
	_, _ = s.bot.Send(msg)
}

// ShowDepositAmountOptions 展示充值金额选项。
func (s *Service) ShowDepositAmountOptions(lang string, callbackQuery *tgbotapi.CallbackQuery) {
	usdtSubscriptionsRepo := repositories.NewUserUSDTSubscriptionsRepository(s.db)
	usdtList, _ := usdtSubscriptionsRepo.ListAvailable(context.Background())

	var allButtons []tgbotapi.InlineKeyboardButton
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, usdtRecord := range usdtList {
		allButtons = append(allButtons, tgbotapi.NewInlineKeyboardButtonData("💰"+usdtRecord.Name, "deposit_usdt_"+usdtRecord.Amount))
	}
	allButtons = append(allButtons, tgbotapi.NewInlineKeyboardButtonData("🔙"+global.Translations[lang]["back_home"], "back_profile"))

	for i := 0; i < len(allButtons)-1; i += 2 {
		end := i + 2
		if end > len(allButtons)-1 {
			end = len(allButtons) - 1
		}
		keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(allButtons[i:end]...))
	}
	keyboard = append(keyboard, tgbotapi.NewInlineKeyboardRow(allButtons[len(allButtons)-1]))

	userRepo := repositories.NewUserRepository(s.db)
	user, _ := userRepo.GetByChatID(callbackQuery.Message.Chat.ID)
	if toolkit.IsEmpty(user.Amount) {
		user.Amount = "0"
	}
	if toolkit.IsEmpty(user.TronAmount) {
		user.TronAmount = "0"
	}

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID,
		"🆔"+global.Translations[lang]["user_id"]+": <code>"+user.Associates+"</code>\n"+
			"👤"+global.Translations[lang]["username"]+": @"+user.Username+"\n"+
			"💰"+global.Translations[lang]["balance"]+": "+"\n"+
			"- TRX：   "+user.TronAmount+"\n"+
			"-  USDT："+user.Amount,
	)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	msg.ParseMode = "HTML"
	s.send(msg)
}

// ShowDepositUSDTRecords 展示 USDT 充值记录。
func (s *Service) ShowDepositUSDTRecords(lang string, callbackQuery *tgbotapi.CallbackQuery) {
	s.renderDepositUSDTPage(lang, callbackQuery, 1)
}

// ShowPrevDepositUSDTPage 展示上一页 USDT 充值记录。
func (s *Service) ShowPrevDepositUSDTPage(lang string, callbackQuery *tgbotapi.CallbackQuery) (*global.DepositState, bool) {
	state := global.DepositStates[callbackQuery.Message.Chat.ID]
	if state != nil && state.CurrentPage == 1 {
		return nil, true
	}
	if state == nil {
		state = &global.DepositState{CurrentPage: 1}
		global.DepositStates[callbackQuery.Message.Chat.ID] = state
	} else {
		state.CurrentPage--
	}
	s.renderDepositUSDTPage(lang, callbackQuery, state.CurrentPage)
	return state, false
}

// ShowNextDepositUSDTPage 展示下一页 USDT 充值记录。
func (s *Service) ShowNextDepositUSDTPage(lang string, callbackQuery *tgbotapi.CallbackQuery) bool {
	state := global.DepositStates[callbackQuery.Message.Chat.ID]
	if state == nil {
		state = &global.DepositState{CurrentPage: 1}
	}
	state.CurrentPage++

	usdtDepositRepo := repositories.NewUserUSDTDepositRepository(s.db)
	var info request.UserUsdtDepositsSearch
	info.PageInfo.Page = state.CurrentPage
	info.PageInfo.PageSize = 10
	_, total, _ := usdtDepositRepo.ListPageByUser(context.Background(), info, callbackQuery.Message.Chat.ID)
	totalPages := (total + info.PageInfo.PageSize - 1) / info.PageInfo.PageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if int64(state.CurrentPage) > totalPages {
		state.CurrentPage = totalPages
		return true
	}

	s.renderDepositUSDTPage(lang, callbackQuery, state.CurrentPage)
	global.DepositStates[callbackQuery.Message.Chat.ID] = state
	return false
}

func (s *Service) resolveParentUID(text string, userRepo *repositories.UserRepository) string {
	index := lastIndex(text, " ")
	if index <= 0 {
		return ""
	}

	parentUID := text[index+1:]
	record, err := userRepo.GetByAssociates(parentUID)
	if err != nil {
		return ""
	}
	return record.Associates
}

func lastIndex(value, sep string) int {
	for i := len(value) - len(sep); i >= 0; i-- {
		if value[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

func (s *Service) renderDepositUSDTPage(lang string, callbackQuery *tgbotapi.CallbackQuery, page int64) {
	usdtDepositRepo := repositories.NewUserUSDTDepositRepository(s.db)
	var info request.UserUsdtDepositsSearch
	info.PageInfo.Page = page
	info.PageInfo.PageSize = 10
	usdtList, _, _ := usdtDepositRepo.ListPageByUser(context.Background(), info, callbackQuery.Message.Chat.ID)

	var builder strings.Builder
	builder.WriteString("\n")
	for _, item := range usdtList {
		builder.WriteString("[")
		builder.WriteString(item.CreatedDate)
		builder.WriteString("]")
		builder.WriteString("+")
		builder.WriteString(item.Amount)
		builder.WriteString(" USDT ")
		builder.WriteString(" （订单 #TOPUP- ")
		builder.WriteString(item.OrderNO)
		builder.WriteString("）")
		builder.WriteString("\n")
	}

	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "🧾"+global.Translations[lang]["deposit_records"]+"\n\n "+
		strings.TrimSpace(builder.String())+"\n")
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(global.Translations[lang]["prev"], "prev_deposit_usdt_page"),
			tgbotapi.NewInlineKeyboardButtonData(global.Translations[lang]["next"], "next_deposit_usdt_page"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙"+global.Translations[lang]["back_home"], "back_profile"),
		),
	)
	s.send(msg)
}

func (s *Service) send(message tgbotapi.Chattable) {
	if _, err := s.bot.Send(message); err != nil {
		logrus.WithError(err).Warn("发送 Telegram 消息失败")
	}
}
