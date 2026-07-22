package order

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"ushield_bot/internal/cache"
	"ushield_bot/internal/config"
	"ushield_bot/internal/domain"
	"ushield_bot/internal/global"
	"ushield_bot/internal/i18n"
	"ushield_bot/internal/infrastructure/repositories"
	toolkit "ushield_bot/internal/infrastructure/tools"
	polyrepo "ushield_bot/internal/poly/repo"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const orderMessageKeySuffix = "_order"

const (
	ProductTopup = "topup"
	ProductData  = "data"
)

// Handler 抽象订单相关能力，便于替换实现和做单测。
type Handler interface {
	ShowDepositUSDTOrder(lang string, callbackQuery *tgbotapi.CallbackQuery)
	CreateTopupOrder(chatID int64, username, mobile, countryID, planID, product string)
	HandleBalancePayment(callbackQuery *tgbotapi.CallbackQuery, orderNo string)
	DeleteCachedOrderMessage(chatID int64)
}

// Service 负责订单生命周期相关能力。
type Service struct {
	cfg   *config.Config
	db    *gorm.DB
	bot   *tgbotapi.BotAPI
	cache cache.Cache
}

// NewService 创建订单服务。
func NewService(cfg *config.Config, db *gorm.DB, bot *tgbotapi.BotAPI, cache cache.Cache) *Service {
	return &Service{
		cfg:   cfg,
		db:    db,
		bot:   bot,
		cache: cache,
	}
}

// ShowDepositUSDTOrder 展示 USDT 充值订单。
func (s *Service) ShowDepositUSDTOrder(lang string, callbackQuery *tgbotapi.CallbackQuery) {
	transferAmount := callbackQuery.Data[13:]

	placeholderRepo := repositories.NewUserUSDTPlaceholdersRepository(s.db)
	placeholder, err := placeholderRepo.GetRandomAvailable(context.Background())
	if err != nil || placeholder.Id == 0 {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, global.Translations[lang]["placeholder_array_size_warning"])
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🕣"+global.Translations[lang]["cancel_order"], "cancel_order"),
				tgbotapi.NewInlineKeyboardButtonData("🔙"+global.Translations[lang]["back_home"], "back_home"),
			),
		)
		msg.ParseMode = "HTML"
		s.send(msg)
		return
	}

	if err := placeholderRepo.UpdateStatusByID(context.Background(), placeholder.Id, 1); err != nil {
		logrus.WithError(err).Warn("锁定 USDT 占位金额失败")
	}

	realTransferAmount := toolkit.AddStringsAsFloats(placeholder.Placeholder, transferAmount)
	depositRepo := repositories.NewUserUSDTDepositRepository(s.db)
	orderNo := toolkit.Generate6DigitOrderNo()
	deposit := domain.UserUSDTDeposits{
		OrderNO:     orderNo,
		UserID:      callbackQuery.Message.Chat.ID,
		Status:      0,
		Placeholder: placeholder.Placeholder,
		Amount:      transferAmount,
		CreatedAt:   time.Now(),
	}

	depositAddress, addressErr := s.resolveDepositAddress(context.Background())
	if addressErr != nil {
		logrus.WithError(addressErr).Warn("查询收款地址失败")
		s.notifyOrderIssue(callbackQuery.Message.Chat.ID, callbackQuery.From.UserName, "USDT充值", addressErr.Error(), "", "")
		return
	}
	deposit.Address = depositAddress

	if err := depositRepo.Create(context.Background(), &deposit); err != nil {
		logrus.WithError(err).Warn("创建 USDT 充值订单失败")
	}

	msg := tgbotapi.NewPhoto(callbackQuery.Message.Chat.ID, tgbotapi.FilePath(s.cfg.OrderImagePath))
	msg.Caption = global.Translations[lang]["order_id"] + "：TOPUP-" + deposit.OrderNO + "\n" +
		global.Translations[lang]["payment_amount"] + "：" + "<code>" + realTransferAmount + "</code>" + " USDT " + global.Translations[lang]["copy_text_tips"] + "\n" +
		global.Translations[lang]["receive_address"] + "<code>" + deposit.Address + "</code>" + global.Translations[lang]["copy_text_tips"] + "\n" +
		global.Translations[lang]["tx_time_limit_tips"] + "\n" +
		global.Translations[lang]["deposit_time_label"] + toolkit.Format4Chinesese(deposit.CreatedAt) + "\n" +
		global.Translations[lang]["amount_suffix_tips"] + "\n"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏳"+global.Translations[lang]["catfee_smart_transaction_pay_button"]+realTransferAmount+" USDT ", "noop"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙"+global.Translations[lang]["back_home"], "back_home"),
			tgbotapi.NewInlineKeyboardButtonData("❌"+global.Translations[lang]["cancel_order"], "cancel_order"),
		),
	)
	msg.ParseMode = "HTML"

	sent, sendErr := s.bot.Send(msg)
	if sendErr != nil {
		logrus.WithError(sendErr).Warn("发送 USDT 充值订单失败")
		return
	}
	s.notifyOrder(callbackQuery.Message.Chat.ID, msg.Caption, "", "")

	expiration := time.Minute
	_ = s.cache.Set(strconv.FormatInt(callbackQuery.Message.Chat.ID, 10)+"_order_no", "USDT_"+deposit.OrderNO, expiration)
	_ = s.cache.Set(s.orderMessageKey(callbackQuery.Message.Chat.ID), strconv.Itoa(sent.MessageID), expiration)
}

// CreateTopupOrder 创建话费或流量充值订单。
func (s *Service) CreateTopupOrder(chatID int64, username, mobile, selectedCountryID, planID, product string) {
	lang := s.resolveUserLang(chatID)
	planName, price, bundleID, planCountryID, countryName, err := s.getPlanInfo(product, planID)
	if err != nil {
		logrus.WithError(err).Warn("查询套餐详情失败")
		return
	}
	orderCountryID, orderCountryName, err := s.resolveOrderCountry(product, selectedCountryID, planCountryID, countryName)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"chat_id":             chatID,
			"plan_id":             planID,
			"bundle_id":           bundleID,
			"selected_country_id": selectedCountryID,
			"plan_country_id":     planCountryID,
			"product":             product,
		}).Warn("校验下单国家失败")
		s.notifyOrderIssue(chatID, username, productLabel(product), err.Error(), s.supportCountryName(selectedCountryID, ""), mobile)
		return
	}

	placeholderRepo := repositories.NewUserUSDTPlaceholdersRepository(s.db)
	placeholder, err := placeholderRepo.GetRandomAvailable(context.Background())
	if err != nil || placeholder.Id == 0 {
		logrus.WithError(err).Warn("查询占位金额失败")
		return
	}
	if err := placeholderRepo.UpdateStatusByID(context.Background(), placeholder.Id, 1); err != nil {
		logrus.WithError(err).Warn("锁定占位金额失败")
	}

	depositAddress, err := s.resolveDepositAddress(context.Background())
	if err != nil {
		logrus.WithError(err).Warn("查询收款地址失败")
		s.notifyOrderIssue(chatID, username, productLabel(product), err.Error(), orderCountryName, mobile)
		return
	}

	orderNo := toolkit.Generate6DigitOrderNo()
	deposit := domain.UserUSDTDeposits{
		OrderNO:     orderNo,
		UserID:      chatID,
		Status:      0,
		Placeholder: placeholder.Placeholder,
		Address:     depositAddress,
		CountryID:   orderCountryID,
		CountryName: orderCountryName,
		Amount:      price,
		CreatedAt:   time.Now(),
		Block:       mobile,
		Source:      orderSource(product),
		BundleId:    bundleID,
	}

	if err := repositories.NewUserUSDTDepositRepository(s.db).Create(context.Background(), &deposit); err != nil {
		logrus.WithError(err).Warn("创建订单失败")
		return
	}

	displayAmount := toolkit.AddStringsAsFloats(price, placeholder.Placeholder)
	tips := i18n.TParam(lang, "purchase_topup", map[string]string{
		"username": username,
		"mobile":   displayMobile(product, mobile, planName),
		"amount":   displayAmount,
		"address":  depositAddress,
	})

	s.notifyOrder(chatID, tips, orderCountryName, mobile)

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(s.cfg.OrderImagePath))
	photo.Caption = tips
	photo.ParseMode = "HTML"
	photo.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⏳"+i18n.T(lang, "balance_pay_order"), "nope_purchase_order"+orderNo),
			tgbotapi.NewInlineKeyboardButtonData(i18n.T(lang, "cancel_order"), backCallback(product)),
		),
	)

	sent, err := s.bot.Send(photo)
	if err != nil {
		logrus.WithError(err).Warn("发送订单图片失败")
		return
	}

	_ = s.cache.Set(s.orderMessageKey(chatID), strconv.Itoa(sent.MessageID), time.Minute)
}

// HandleBalancePayment 处理余额支付。
func (s *Service) HandleBalancePayment(callbackQuery *tgbotapi.CallbackQuery, orderNo string) {
	lang := s.resolveUserLang(callbackQuery.Message.Chat.ID)
	depositRepo := repositories.NewUserUSDTDepositRepository(s.db)
	record, err := depositRepo.GetByOrderNo(context.Background(), orderNo)
	if err != nil {
		logrus.WithError(err).Warn("查询支付订单失败")
		return
	}

	userRepo := repositories.NewUserRepository(s.db)
	user, err := userRepo.GetByChatID(callbackQuery.Message.Chat.ID)
	if err != nil {
		logrus.WithError(err).Warn("查询用户余额失败")
		return
	}
	if toolkit.IsEmpty(user.Amount) {
		user.Amount = "0"
	}

	if flag, _ := toolkit.CompareNumberStrings(user.Amount, record.Amount); flag < 0 {
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID,
			"<b>"+"🔍"+i18n.T(lang, "insufficient_balance_tips")+"</b>"+"\n"+
				"🆔"+i18n.T(lang, "user_id")+": <code>"+user.Associates+"</code>\n"+
				"👤"+i18n.T(lang, "username")+": @"+user.Username+"\n"+
				"💰"+i18n.T(lang, "balance")+"\n"+
				"-  USDT："+user.Amount)
		msg.ParseMode = "HTML"
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💵"+i18n.T(lang, "deposit"), "deposit_amount"),
			),
		)
		s.send(msg)
		return
	}

	balance, _ := toolkit.SubtractStringNumbers(user.Amount, record.Amount, 1)
	user.Amount = balance
	if err := userRepo.Save(context.Background(), &user); err != nil {
		logrus.WithError(err).Warn("扣减余额失败")
		return
	}

	placeholderRepo := repositories.NewUserUSDTPlaceholdersRepository(s.db)
	if err := placeholderRepo.UpdateStatusByPlaceholder(context.Background(), record.Placeholder, 0); err != nil {
		logrus.WithError(err).Warn("释放占位金额失败")
	}
	if err := depositRepo.UpdateStatusByOrderNo(context.Background(), orderNo, 2); err != nil {
		logrus.WithError(err).Warn("更新订单状态失败")
	}

	tips := i18n.TParam(lang, "successfully_purchased_order", map[string]string{
		"amount":  record.Amount,
		"support": s.supportUsername(lang),
	})
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, i18n.T(lang, "order_id")+"：TOPUP-"+orderNo+"\n"+tips)
	msg.ParseMode = "HTML"
	s.send(msg)
	s.notifyBalancePaymentSuccess(callbackQuery.Message.Chat.ID, user.Username, orderNo, record)
}

// DeleteCachedOrderMessage 删除缓存中的历史订单消息。
func (s *Service) DeleteCachedOrderMessage(chatID int64) {
	prevMessageIDStr, _ := s.cache.Get(s.orderMessageKey(chatID))
	if prevMessageIDStr == "" {
		return
	}

	prevMessageID, err := strconv.Atoi(prevMessageIDStr)
	if err != nil {
		return
	}

	if _, err := s.bot.Request(tgbotapi.DeleteMessageConfig{ChatID: chatID, MessageID: prevMessageID}); err != nil {
		logrus.WithError(err).Warn("删除历史订单消息失败")
	}
}

func (s *Service) orderMessageKey(chatID int64) string {
	return strconv.FormatInt(chatID, 10) + orderMessageKeySuffix
}

func (s *Service) send(message tgbotapi.Chattable) {
	if _, err := s.bot.Send(message); err != nil {
		logrus.WithError(err).Warn("发送 Telegram 消息失败")
	}
}

func (s *Service) resolveUserLang(chatID int64) string {
	lang, err := s.cache.Get("LANG_" + strconv.FormatInt(chatID, 10))
	if err == nil && lang != "" {
		return lang
	}

	userRepo := repositories.NewUserRepository(s.db)
	record, repoErr := userRepo.GetByChatID(chatID)
	if repoErr == nil && record.Lang != "" {
		return record.Lang
	}
	if global.DefaultLang != "" {
		return global.DefaultLang
	}
	return "zh"
}

func (s *Service) notifyOrder(chatID int64, tips, countryName, mobile string) {
	if s.cfg.NotifyChatID == 0 {
		return
	}

	msg := tgbotapi.NewMessage(
		s.cfg.NotifyChatID,
		tips+"\n ID: "+strconv.FormatInt(chatID, 10)+
			"\n国家："+supportValue(countryName, "未知国家")+
			"\n号码："+supportValue(mobile, "未知号码")+
			"\n\n<b>状态：支付中</b>",
	)
	msg.ParseMode = "HTML"
	s.send(msg)
}

func (s *Service) notifyOrderIssue(chatID int64, username, scene, reason, countryName, mobile string) {
	if s.cfg.NotifyChatID == 0 {
		return
	}

	userDisplay := "@unknown"
	if username != "" {
		userDisplay = "@" + username
	}
	msg := tgbotapi.NewMessage(
		s.cfg.NotifyChatID,
		"⚠️<b>下单异常告警</b>\n"+
			"场景："+scene+"\n"+
			"用户ID："+strconv.FormatInt(chatID, 10)+"\n"+
			"用户名："+userDisplay+"\n"+
			"国家："+supportValue(countryName, "未知国家")+"\n"+
			"号码："+supportValue(mobile, "未知号码")+"\n"+
			"原因："+reason,
	)
	msg.ParseMode = "HTML"
	s.send(msg)
}

func (s *Service) notifyBalancePaymentSuccess(chatID int64, username, orderNo string, record domain.UserUSDTDeposits) {
	if s.cfg.NotifyChatID == 0 {
		return
	}

	userDisplay := "@unknown"
	if username != "" {
		userDisplay = "@" + username
	}
	countryName := record.CountryName
	mobile := record.Block
	msg := tgbotapi.NewMessage(
		s.cfg.NotifyChatID,
		"✅<b>余额支付成功</b>\n"+
			"订单号：TOPUP-"+orderNo+"\n"+
			"用户ID："+strconv.FormatInt(chatID, 10)+"\n"+
			"用户名："+userDisplay+"\n"+
			"国家："+supportValue(countryName, "未知国家")+"\n"+
			"号码："+supportValue(mobile, "未知号码")+"\n"+
			"金额："+record.Amount+" USDT",
	)
	msg.ParseMode = "HTML"
	s.send(msg)
}

func (s *Service) supportCountryName(countryID, fallback string) string {
	if fallback != "" {
		return fallback
	}
	countryName, err := s.resolveCountryName(countryID)
	if err != nil {
		logrus.WithError(err).WithField("country_id", countryID).Warn("查询客服通知国家名称失败")
		return ""
	}
	return countryName
}

func supportValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func (s *Service) resolveDepositAddress(ctx context.Context) (string, error) {
	agentName := ""
	if s.cfg != nil {
		agentName = s.cfg.AgentName
	}
	if agentName == "" {
		agentName = os.Getenv("AGENT")
	}
	if agentName == "" {
		return "", fmt.Errorf("缺少 AGENT 配置")
	}

	_, depositAddress, err := repositories.NewSystemUserRepository(s.db).GetAddressesByUsername(ctx, agentName)
	if err != nil {
		return "", err
	}
	if depositAddress == "" {
		return "", fmt.Errorf("用户 %s 的收款地址为空", agentName)
	}
	return depositAddress, nil
}

func (s *Service) getPlanInfo(product, planID string) (string, string, int64, int64, string, error) {
	switch product {
	case ProductTopup:
		item, err := polyrepo.NewExpensesTopUpPlanRepo(s.db).Get(context.Background(), planID)
		if err != nil {
			return "", "", 0, 0, "", err
		}
		countryName, countryErr := s.resolveCountryName(strconv.Itoa(item.CountryID))
		if countryErr != nil {
			logrus.WithError(countryErr).WithFields(logrus.Fields{
				"product":    product,
				"plan_id":    planID,
				"country_id": item.CountryID,
			}).Warn("查询国家名称失败，按空值继续下单")
		}
		return item.NameCn, item.Price, int64(item.ID), int64(item.CountryID), countryName, nil
	case ProductData:
		item, err := polyrepo.NewDataTopUpPlanRepo(s.db).Get(context.Background(), planID)
		if err != nil {
			return "", "", 0, 0, "", err
		}
		countryName, countryErr := s.resolveCountryName(strconv.Itoa(item.CountryID))
		if countryErr != nil {
			logrus.WithError(countryErr).WithFields(logrus.Fields{
				"product":    product,
				"plan_id":    planID,
				"country_id": item.CountryID,
			}).Warn("查询国家名称失败，按空值继续下单")
		}
		return item.NameCn, item.Price, int64(item.ID), int64(item.CountryID), countryName, nil
	default:
		return "", "", 0, 0, "", fmt.Errorf("未知充值类型: %s", product)
	}
}

func (s *Service) resolveCountryName(countryID string) (string, error) {
	if countryID == "" || countryID == "0" {
		return "", nil
	}
	country, err := polyrepo.NewCountryRepo(s.db).Get(context.Background(), countryID)
	if err != nil {
		return "", err
	}
	if country.NameCn != "" {
		return country.NameCn, nil
	}
	if country.NameEn != "" {
		return country.NameEn, nil
	}
	return "", nil
}

func (s *Service) resolveOrderCountry(product, selectedCountryID string, planCountryID int64, planCountryName string) (int64, string, error) {
	selectedCountryID = strings.TrimSpace(selectedCountryID)
	if selectedCountryID == "" {
		return 0, "", fmt.Errorf("传入 countryID 为空")
	}

	selectedID, err := strconv.ParseInt(selectedCountryID, 10, 64)
	if err != nil || selectedID <= 0 {
		return 0, "", fmt.Errorf("传入 countryID 非法: %s", selectedCountryID)
	}
	if product == ProductData {
		countryName, resolveErr := s.resolveCountryName(strconv.FormatInt(selectedID, 10))
		if resolveErr != nil {
			logrus.WithError(resolveErr).WithFields(logrus.Fields{
				"selected_country_id": selectedID,
				"product":             product,
			}).Warn("按原始点击路径查询流量国家名称失败")
			return selectedID, "", nil
		}
		return selectedID, countryName, nil
	}
	if planCountryID <= 0 {
		countryName := planCountryName
		if countryName == "" {
			resolvedName, resolveErr := s.resolveCountryName(strconv.FormatInt(selectedID, 10))
			if resolveErr != nil {
				logrus.WithError(resolveErr).WithFields(logrus.Fields{
					"selected_country_id": selectedID,
				}).Warn("套餐 country_id 缺失，回退选中国家时查询名称失败")
			} else {
				countryName = resolvedName
			}
		}
		return selectedID, countryName, nil
	}
	if selectedID != planCountryID {
		return 0, "", fmt.Errorf("countryID 不一致: 传入=%d, 套餐=%d", selectedID, planCountryID)
	}
	return planCountryID, planCountryName, nil
}

func displayMobile(product, mobile, planName string) string {
	if product == ProductData {
		return mobile + "  " + planName
	}
	return mobile
}

func orderSource(product string) int64 {
	if product == ProductData {
		return 10
	}
	return 9
}

func backCallback(product string) string {
	if product == ProductData {
		return "back_home_data"
	}
	return "back_home"
}

func productLabel(product string) string {
	if product == ProductData {
		return "流量充值"
	}
	return "话费充值"
}

func (s *Service) supportUsername(lang string) string {
	if s.cfg != nil && strings.TrimSpace(s.cfg.SupportUsername) != "" {
		return strings.TrimSpace(s.cfg.SupportUsername)
	}
	return i18n.T(lang, "support")
}
