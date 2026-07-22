package topup

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ushield_bot/internal/cache"
	"ushield_bot/internal/global"
	"ushield_bot/internal/poly/model"
	polyrepo "ushield_bot/internal/poly/repo"
	orderservice "ushield_bot/internal/service/order"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	StateSetReminderPrefix = "setReminder_"
	StateTopupMobilePrefix = "click_mobile_topup"
	StateDataMobilePrefix  = "click_mobile_data_topup"
	ActionAddMobile        = "add_mobile_mgr"
	ActionDeleteMobile     = "delete_mobile_mgr"
)

// Handler 抽象话费/流量充值相关能力，便于替换实现和测试。
type Handler interface {
	PromptReminderDay(chatID int64, reminderID string)
	HandleReminderDayInput(chatID int64, input, reminderID string)
	ShowCountryMenu(chatID int64, product string)
	ShowPlanMenu(chatID int64, countryID, product string)
	ShowPlanMobilePrompt(chatID int64, countryID, planID, product string)
	HandleMobileInput(chatID int64, username, input, countryID, planID, product string)
	PromptMobileManagerInput(chatID int64, countryID, action string)
	HandleAddMobileInput(chatID int64, input, countryID string)
	HandleDeleteMobileInput(chatID int64, input, countryID string)
	ShowMobileManager(chatID int64, countryID string, showDefaultReminder bool)
}

// Service 负责话费与流量充值链路。
type Service struct {
	db           *gorm.DB
	bot          *tgbotapi.BotAPI
	cache        cache.Cache
	orderService *orderservice.Service
}

// NewService 创建充值服务。
func NewService(db *gorm.DB, bot *tgbotapi.BotAPI, cache cache.Cache, orderService *orderservice.Service) *Service {
	return &Service{
		db:           db,
		bot:          bot,
		cache:        cache,
		orderService: orderService,
	}
}

// PromptReminderDay 展示提醒日输入提示。
func (s *Service) PromptReminderDay(chatID int64, reminderID string) {
	mobileRepo := polyrepo.NewPolytoupUserMobileRepository(s.db)
	mobileInfo, err := mobileRepo.Get(context.Background(), reminderID)
	if err != nil {
		logrus.WithError(err).Warn("查询提醒号码失败")
		return
	}
	if mobileInfo.ChatID != chatID {
		logrus.WithField("chat_id", chatID).Warn("无权限设置提醒")
		return
	}

	msg := tgbotapi.NewMessage(chatID, "您希望每月几号收到充值提醒？请输入1至28之间的日期。\n\n如不填写，系统将默认在每月1号为您发送提醒")
	msg.ParseMode = "HTML"
	s.send(msg)
	_ = s.cache.Set(s.stateKey(chatID), StateSetReminderPrefix+strconv.Itoa(int(mobileInfo.ID)), time.Minute)
}

// HandleReminderDayInput 处理提醒日输入。
func (s *Service) HandleReminderDayInput(chatID int64, input, reminderID string) {
	day, err := strconv.ParseInt(strings.TrimSpace(input), 10, 64)
	if err != nil || day < 1 || day > 28 {
		msg := tgbotapi.NewMessage(chatID, "当前设置时间有误，无法添加，请重新输入\n")
		msg.ParseMode = "HTML"
		s.send(msg)
		_ = s.cache.Set(s.stateKey(chatID), StateSetReminderPrefix+reminderID, time.Minute)
		return
	}

	mobileRepo := polyrepo.NewPolytoupUserMobileRepository(s.db)
	if err := mobileRepo.UpdateReminderDay(context.Background(), reminderID, day); err != nil {
		logrus.WithError(err).Warn("更新提醒日失败")
		return
	}

	successMsg := tgbotapi.NewMessage(chatID, "✅<b>提醒日，设置成功</b>\n")
	successMsg.ParseMode = "HTML"
	s.send(successMsg)

	mobileInfo, err := mobileRepo.Query(context.Background(), reminderID)
	if err != nil {
		logrus.WithError(err).Warn("查询提醒号码失败")
		return
	}

	s.ShowMobileManager(chatID, strconv.Itoa(mobileInfo.CountryID), false)
}

// ShowCountryMenu 展示国家选择菜单。
func (s *Service) ShowCountryMenu(chatID int64, product string) {
	countryItems, err := polyrepo.NewCountryRepo(s.db).List(context.Background())
	if err != nil {
		logrus.WithError(err).Warn("查询国家列表失败")
		return
	}

	buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(countryItems))
	for _, countryItem := range countryItems {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(countryItem.NameCn, countryCallbackPrefix(product)+strconv.FormatUint(uint64(countryItem.ID), 10)))
	}

	msg := tgbotapi.NewMessage(chatID, countryPrompt(product))
	msg.ReplyMarkup = buildInlineKeyboard(buttons, 4)
	msg.ParseMode = "HTML"
	s.send(msg)
}

// ShowPlanMenu 展示套餐选择菜单。
func (s *Service) ShowPlanMenu(chatID int64, countryID, product string) {
	planButtons, err := s.planButtons(product, countryID)
	if err != nil {
		logrus.WithError(err).Warn("查询充值套餐失败")
		return
	}

	extraButtons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔢号码管理", "click_mobile_mgr"+countryID),
		tgbotapi.NewInlineKeyboardButtonData("🔙️"+global.Translations["zh"]["back_homepage"], backCallback(product)),
	}

	keyboard := append(buildInlineKeyboardRows(planButtons, 2), tgbotapi.NewInlineKeyboardRow(extraButtons...))
	msg := tgbotapi.NewMessage(chatID, planPrompt(product))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	msg.ParseMode = "HTML"
	s.send(msg)
}

// ShowPlanMobilePrompt 展示号码输入或已保存号码选择菜单。
func (s *Service) ShowPlanMobilePrompt(chatID int64, countryID, planID, product string) {
	country, _ := polyrepo.NewCountryRepo(s.db).Get(context.Background(), countryID)
	mobileRepo := polyrepo.NewPolytoupUserMobileRepository(s.db)
	items, err := mobileRepo.ListAll(context.Background(), countryID, strconv.FormatInt(chatID, 10))
	if err != nil {
		logrus.WithError(err).Warn("查询号码列表失败")
	}

	if len(items) == 0 {
		msg := tgbotapi.NewMessage(chatID, mobileInputPrompt(product, country))
		msg.ParseMode = "HTML"
		s.send(msg)
		_ = s.cache.Set(s.stateKey(chatID), inputStatePrefix(product)+buildSelectionPayload(countryID, planID), time.Minute)
		return
	}

	buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(items))
	for _, item := range items {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(item.Mobile, mobileSelectPrefix(product)+item.Mobile+"="+buildSelectionPayload(countryID, planID)))
	}

	msg := tgbotapi.NewMessage(chatID, mobileSavedPrompt(product, country))
	msg.ReplyMarkup = buildInlineKeyboard(buttons, 1)
	msg.ParseMode = "HTML"
	s.send(msg)
	_ = s.cache.Set(s.stateKey(chatID), inputStatePrefix(product)+buildSelectionPayload(countryID, planID), time.Minute)
}

// HandleMobileInput 处理号码输入并创建订单。
func (s *Service) HandleMobileInput(chatID int64, username, input, countryID, planID, product string) {
	mobile := strings.TrimSpace(input)
	if !validMobileEntry(mobile) {
		msg := tgbotapi.NewMessage(chatID, invalidMobilePrompt())
		msg.ParseMode = "HTML"
		s.send(msg)
		return
	}

	s.orderService.CreateTopupOrder(chatID, username, mobile, countryID, planID, product)
}

// PromptMobileManagerInput 提示用户输入待新增或待删除号码。
func (s *Service) PromptMobileManagerInput(chatID int64, countryID, action string) {
	country, _ := polyrepo.NewCountryRepo(s.db).Get(context.Background(), countryID)

	var content string
	switch action {
	case ActionAddMobile:
		content = "请务必输入有效的 <b>" + country.NameCn + "</b> 手机号码。并包含国家区号 （正确格式示例：+86 123456789）因号码错误所导致的充值失败，本公司概不负责: "
	case ActionDeleteMobile:
		content = "请输入需要删除的 <b>" + country.NameCn + "</b> 手机号码。并包含国家区号 （正确格式示例：+86 123456789）因号码错误所导致的充值失败，本公司概不负责: "
	default:
		return
	}

	msg := tgbotapi.NewMessage(chatID, content)
	msg.ParseMode = "HTML"
	s.send(msg)
	_ = s.cache.Set(s.stateKey(chatID), action+countryID, time.Minute)
}

// HandleAddMobileInput 处理新增号码。
func (s *Service) HandleAddMobileInput(chatID int64, input, countryID string) {
	mobile := strings.TrimSpace(input)
	if !validMobileEntry(mobile) {
		msg := tgbotapi.NewMessage(chatID, invalidMobilePrompt())
		msg.ParseMode = "HTML"
		s.send(msg)
		return
	}

	mobileRepo := polyrepo.NewPolytoupUserMobileRepository(s.db)
	if count := mobileRepo.Count(context.Background(), countryID, chatID); count > 10 {
		country, _ := polyrepo.NewCountryRepo(s.db).Get(context.Background(), countryID)
		msg := tgbotapi.NewMessage(chatID, "当前 <b>"+country.NameCn+"</b> 手机号码列表已经大于10个，无法添加\n")
		msg.ParseMode = "HTML"
		s.send(msg)
		return
	}

	countryNumericID, _ := strconv.Atoi(countryID)
	pkg := model.UserMobile{
		ALI_MODEL: model.ALI_MODEL{
			CreatedAt: time.Now(),
		},
		CountryID:   countryNumericID,
		ChatID:      chatID,
		Status:      1,
		Mobile:      mobile,
		ReminderDay: "1",
	}
	if err := mobileRepo.Create(context.Background(), &pkg); err != nil {
		logrus.WithError(err).Warn("添加号码失败")
		return
	}

	msg := tgbotapi.NewMessage(chatID, "✅<b>手机号码，添加成功</b>\n")
	msg.ParseMode = "HTML"
	s.send(msg)

	s.ShowMobileManager(chatID, countryID, true)
}

// HandleDeleteMobileInput 处理删除号码。
func (s *Service) HandleDeleteMobileInput(chatID int64, input, countryID string) {
	mobileRepo := polyrepo.NewPolytoupUserMobileRepository(s.db)
	if err := mobileRepo.Delete2(context.Background(), countryID, strings.TrimSpace(input)); err != nil {
		logrus.WithError(err).Warn("删除号码失败")
		return
	}

	msg := tgbotapi.NewMessage(chatID, "✅<b>手机号码，删除成功</b>\n")
	msg.ParseMode = "HTML"
	s.send(msg)

	s.ShowMobileManager(chatID, countryID, false)
}

// ShowMobileManager 展示号码管理面板。
func (s *Service) ShowMobileManager(chatID int64, countryID string, showDefaultReminder bool) {
	country, _ := polyrepo.NewCountryRepo(s.db).Get(context.Background(), countryID)
	mobileRepo := polyrepo.NewPolytoupUserMobileRepository(s.db)
	items, err := mobileRepo.ListAll(context.Background(), countryID, strconv.FormatInt(chatID, 10))
	if err != nil {
		logrus.WithError(err).Warn("查询号码管理列表失败")
	}

	msg := tgbotapi.NewMessage(chatID, buildMobileManagerText(country.NameCn, items, showDefaultReminder))
	msg.ParseMode = "HTML"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("➕添加号码", ActionAddMobile+countryID),
			tgbotapi.NewInlineKeyboardButtonData("➖删除号码", ActionDeleteMobile+countryID),
		),
	)
	s.send(msg)
}

func (s *Service) stateKey(chatID int64) string {
	return strconv.FormatInt(chatID, 10)
}

func (s *Service) send(message tgbotapi.Chattable) {
	if _, err := s.bot.Send(message); err != nil {
		logrus.WithError(err).Warn("发送 Telegram 消息失败")
	}
}

func (s *Service) planButtons(product, countryID string) ([]tgbotapi.InlineKeyboardButton, error) {
	switch product {
	case orderservice.ProductTopup:
		items, err := polyrepo.NewExpensesTopUpPlanRepo(s.db).List(context.Background())
		if err != nil {
			return nil, err
		}
		buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(items))
		for _, item := range items {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(item.NameCn, planCallbackPrefix(product)+countryID+"="+strconv.FormatUint(uint64(item.ID), 10)))
		}
		return buttons, nil
	case orderservice.ProductData:
		items, err := polyrepo.NewDataTopUpPlanRepo(s.db).List(context.Background())
		if err != nil {
			return nil, err
		}
		buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(items))
		for _, item := range items {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(item.NameCn, planCallbackPrefix(product)+countryID+"="+strconv.FormatUint(uint64(item.ID), 10)))
		}
		return buttons, nil
	default:
		return nil, fmt.Errorf("未知充值类型: %s", product)
	}
}

func buildMobileManagerText(countryName string, items []model.UserMobile, showDefaultReminder bool) string {
	var builder strings.Builder
	builder.WriteString("当前 <b>")
	builder.WriteString(countryName)
	builder.WriteString("</b> 手机号码列表: \n")
	if showDefaultReminder {
		builder.WriteString("⚠️默认每个月1号会提醒充值\n")
	}

	for _, item := range items {
		builder.WriteString("\n<code>")
		builder.WriteString(item.Mobile)
		builder.WriteString("</code> 提醒")
		builder.WriteString(item.ReminderDay)
		builder.WriteString("号\n")
		builder.WriteString("设置提醒日:/setReminder_")
		builder.WriteString(strconv.Itoa(int(item.ID)))
		builder.WriteString("\n➖➖➖➖➖➖➖➖➖➖➖➖➖")
	}

	return strings.TrimSpace(builder.String()) + "\n"
}

func buildInlineKeyboard(buttons []tgbotapi.InlineKeyboardButton, perRow int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(buildInlineKeyboardRows(buttons, perRow)...)
}

func buildInlineKeyboardRows(buttons []tgbotapi.InlineKeyboardButton, perRow int) [][]tgbotapi.InlineKeyboardButton {
	if perRow <= 0 {
		perRow = 1
	}

	rows := make([][]tgbotapi.InlineKeyboardButton, 0, (len(buttons)+perRow-1)/perRow)
	for i := 0; i < len(buttons); i += perRow {
		end := i + perRow
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(buttons[i:end]...))
	}
	return rows
}

func validMobileEntry(mobile string) bool {
	entry := strings.TrimSpace(mobile)
	if entry == "" {
		return false
	}

	parts := strings.Fields(entry)
	operator := ""
	number := ""
	switch {
	case len(parts) >= 2:
		operator = strings.Join(parts[:len(parts)-1], " ")
		number = parts[len(parts)-1]
	case strings.Contains(entry, "+"):
		plusIndex := strings.LastIndex(entry, "+")
		if plusIndex <= 0 {
			return false
		}
		operator = strings.TrimSpace(entry[:plusIndex])
		number = strings.TrimSpace(entry[plusIndex:])
	default:
		return false
	}

	if len(operator) == 0 || len(operator) > 20 {
		return false
	}
	if len(number) < 5 || len(number) > 20 {
		return false
	}
	if !strings.HasPrefix(number, "+") {
		return false
	}
	for _, ch := range number[1:] {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func countryCallbackPrefix(product string) string {
	if product == orderservice.ProductData {
		return "click_data_"
	}
	return "click_country_"
}

func planCallbackPrefix(product string) string {
	if product == orderservice.ProductData {
		return "click_5G_data_"
	}
	return "click_plan_"
}

func mobileSelectPrefix(product string) string {
	if product == orderservice.ProductData {
		return StateDataMobilePrefix
	}
	return StateTopupMobilePrefix
}

func inputStatePrefix(product string) string {
	if product == orderservice.ProductData {
		return StateDataMobilePrefix
	}
	return StateTopupMobilePrefix
}

func buildSelectionPayload(countryID, planID string) string {
	return countryID + "=" + planID
}

func backCallback(product string) string {
	if product == orderservice.ProductData {
		return "back_home_data"
	}
	return "back_home"
}

func countryPrompt(product string) string {
	if product == orderservice.ProductData {
		return "请选择充值流量国家\n工作时间：9:00-22:00\n📌1、流量无法标准化，各个国家流量期限不一致。\n📌2、遇问题请第一时间联系客服。\n📌3、具体规则以客服解释执行为准。"
	}
	return "请选择充值话费国家\n工作时间：9:00-22:00"
}

func planPrompt(product string) string {
	if product == orderservice.ProductData {
		return "请选择5G流量套餐:"
	}
	return "请选择充值金额\n✅ 按实时汇率自动结算\n✅ 充值成功后，将收取 5% 服务佣金（仅在到账金额中扣除）"
}

func mobileSavedPrompt(product string, country model.Country) string {
	if product == orderservice.ProductData {
		return "请点击下方号码充值，或重新输入您要充值的<b>" + country.NameCn + "</b>" + country.NameEn + "，运营商 + 手机号码（示例：移动 +86138123456789）: "
	}
	return "请点击下方号码充值，或重新输入您要充值的<b>" + country.NameCn + "</b>运营商 + 手机号码（示例：移动 +86138123456789）: "
}

func mobileInputPrompt(product string, country model.Country) string {
	if product == orderservice.ProductData {
		return "请输入您要充值的<b>" + country.NameCn + "</b>" + country.NameEn + "，运营商 + 手机号码（示例：移动 +86138123456789）: "
	}
	return "请输入您要充值的<b>" + country.NameCn + "</b>运营商 + 手机号码（示例：移动 +86138123456789）: "
}

func invalidMobilePrompt() string {
	return "当前号码格式有误，请按“运营商 + 手机号码”或“运营商+手机号码”格式输入\n示例：移动 +86138123456789 或 移动+86138123456789\n"
}
