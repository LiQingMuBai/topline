package testsupport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/require"
	"ushield_bot/internal/global"
)

// TelegramRequest 记录一次发往 Telegram API 的测试请求。
type TelegramRequest struct {
	Method string
	Form   url.Values
}

// TelegramRecorder 用于在测试中断言发出的 Telegram 请求。
type TelegramRecorder struct {
	mu       sync.Mutex
	requests []TelegramRequest
}

// Requests 返回请求快照，避免测试中读写竞争。
func (r *TelegramRecorder) Requests() []TelegramRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]TelegramRequest, 0, len(r.requests))
	for _, item := range r.requests {
		result = append(result, TelegramRequest{
			Method: item.Method,
			Form:   cloneValues(item.Form),
		})
	}
	return result
}

func (r *TelegramRecorder) append(method string, form url.Values) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests = append(r.requests, TelegramRequest{
		Method: method,
		Form:   cloneValues(form),
	})
}

// NewTestBot 创建一个指向本地 httptest Server 的 BotAPI。
func NewTestBot(t *testing.T) (*tgbotapi.BotAPI, *TelegramRecorder, func()) {
	t.Helper()

	recorder := &TelegramRecorder{}
	messageID := 100
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			_ = r.ParseMultipartForm(8 << 20)
		} else {
			_ = r.ParseForm()
		}

		method := path.Base(r.URL.Path)
		if method != "getMe" {
			recorder.append(method, r.PostForm)
		}
		w.Header().Set("Content-Type", "application/json")

		switch method {
		case "getMe":
			writeJSON(w, map[string]any{
				"ok": true,
				"result": map[string]any{
					"id":         1,
					"is_bot":     true,
					"first_name": "test",
					"username":   "test_bot",
				},
			})
		case "sendMessage", "sendPhoto":
			messageID++
			chatID, _ := strconv.ParseInt(r.PostForm.Get("chat_id"), 10, 64)
			writeJSON(w, map[string]any{
				"ok": true,
				"result": map[string]any{
					"message_id": messageID,
					"date":       0,
					"chat": map[string]any{
						"id":   chatID,
						"type": "private",
					},
				},
			})
		default:
			writeJSON(w, map[string]any{
				"ok":     true,
				"result": true,
			})
		}
	}))

	bot, err := tgbotapi.NewBotAPIWithClient("test-token", server.URL+"/bot%s/%s", server.Client())
	require.NoError(t, err)

	cleanup := func() {
		server.Close()
	}

	return bot, recorder, cleanup
}

// SeedTranslations 写入测试需要的最小中文翻译集。
func SeedTranslations() {
	global.DefaultLang = "zh"
	global.SupportedLangs = []string{"zh", "en"}
	global.Translations["zh"] = map[string]string{
		"insufficient_balance_tips":       "余额不足，请先充值",
		"user_id":                         "用户ID",
		"username":                        "用户名",
		"balance":                         "余额",
		"deposit":                         "充值",
		"order_id":                        "订单号",
		"successfully_purchased_order":    "✅成功购买价值{amount}USDT的话费或流量，请耐心等待帮您充值，有任何问题，请联系客服 {support}",
		"continent_africa":                "🌍 非洲",
		"continent_america":               "🌎 美洲",
		"continent_asia":                  "🌏 亚洲",
		"continent_europe":                "🌍 欧洲",
		"continent_oceania":               "🌏 大洋洲",
		"continent_other":                 "🌐 其他",
		"deposit_records":                 "充值记录",
		"prev":                            "上一页",
		"next":                            "下一页",
		"back_home":                       "返回个人中心",
		"back_homepage":                   "返回首页",
		"support":                         "客服",
		"menu_topup":                      "⛽话费充值",
		"menu_data":                       "🔋流量充值",
		"menu_profile":                    "👤个人中心",
		"menu_support":                    "🐍联系娘子",
		"menu_language":                   "🌐切换语言",
		"welcome_tips":                    "欢迎使用中文菜单",
		"hide_keyboard_tips":              "键盘已隐藏，发送 /start 重新显示",
		"language_menu_title":             "请选择语言 / Please choose a language",
		"language_option_zh":              "中文",
		"language_option_en":              "English",
		"language_switched":               "语言已切换成功。",
		"support_message":                 "📞{support}\n工作时间：{work_time}\n",
		"support_work_time_line":          "工作时间：{work_time}",
		"topup_country_prompt_data":       "请选择充值流量国家{work_time_line}\n📌1、流量无法标准化，各个国家流量期限不一致。\n📌2、遇问题请第一时间联系客服。\n📌3、具体规则以客服解释执行为准。",
		"topup_country_prompt_topup":      "请选择充值话费国家{work_time_line}",
		"topup_mobile_input_prompt_data":  "请输入您要充值的<b>{country}</b>运营商 + 手机号码（示例：移动 +86138123456789）: ",
		"topup_mobile_input_prompt_topup": "请输入您要充值的<b>{country}</b>运营商 + 手机号码（示例：移动 +86138123456789）: ",
		"topup_mobile_saved_prompt_data":  "请点击下方号码充值，或重新输入您要充值的<b>{country}</b>运营商 + 手机号码（示例：移动 +86138123456789）: ",
		"topup_mobile_saved_prompt_topup": "请点击下方号码充值，或重新输入您要充值的<b>{country}</b>运营商 + 手机号码（示例：移动 +86138123456789）: ",
		"topup_mobile_manager_button":     "🔢号码管理",
		"topup_add_mobile_button":         "➕添加号码",
		"topup_delete_mobile_button":      "➖删除号码",
		"topup_plan_prompt_data":          "请选择5G流量套餐:",
		"topup_plan_prompt_topup":         "请选择充值金额\n✅ 按实时汇率自动结算\n✅ 充值成功后，将收取 {fee_rate} 服务佣金（仅在到账金额中扣除）",
		"purchase_topup":                  "⚠️↓↓请按金额支付，否则无法到账↓↓\n---------------------------------\n🔸用户账号：@{username}\n🔸手机号码：<code>{mobile}</code> \n🔸支付金额：<code>{amount}</code> USDT\n🔸收款地址：<code>{address}</code>\n（点击地址复制）\n---------------------------------\n\n‼️请务必核对金额尾数，金额带小数\n",
		"balance_pay_order":               "余额支付",
		"cancel_order":                    "取消订单",
	}
	global.Translations["en"] = map[string]string{
		"insufficient_balance_tips":       "Insufficient balance. Please top up first.",
		"user_id":                         "User ID",
		"username":                        "Username",
		"balance":                         "Balance",
		"deposit":                         "Deposit",
		"order_id":                        "Order ID",
		"successfully_purchased_order":    "Successfully purchased {amount} USDT worth of top-up or data. Contact support {support}.",
		"continent_africa":                "🌍 Africa",
		"continent_america":               "🌎 Americas",
		"continent_asia":                  "🌏 Asia",
		"continent_europe":                "🌍 Europe",
		"continent_oceania":               "🌏 Oceania",
		"continent_other":                 "🌐 Other",
		"deposit_records":                 "Deposit Records",
		"prev":                            "Previous",
		"next":                            "Next",
		"back_home":                       "Back to Profile",
		"back_homepage":                   "Back to Home",
		"support":                         "Support",
		"menu_topup":                      "⛽ Top Up",
		"menu_data":                       "🔋 Data Top Up",
		"menu_profile":                    "👤 Profile",
		"menu_support":                    "🐍 Support",
		"menu_language":                   "🌐 Language",
		"welcome_tips":                    "Welcome to the English menu",
		"hide_keyboard_tips":              "Keyboard hidden. Send /start to show it again.",
		"language_menu_title":             "Please choose a language",
		"language_option_zh":              "中文",
		"language_option_en":              "English",
		"language_switched":               "Language switched successfully.",
		"support_message":                 "📞{support}\nWorking hours: {work_time}\n",
		"support_work_time_line":          "Working hours: {work_time}",
		"topup_country_prompt_data":       "Please select a data top-up country{work_time_line}\n📌1. Data packages are not standardized and validity varies by country.\n📌2. If you encounter any issues, please contact support immediately.\n📌3. Final rules are subject to support explanation.",
		"topup_country_prompt_topup":      "Please select a top-up country{work_time_line}",
		"topup_mobile_input_prompt_data":  "Please enter <b>{country}</b> carrier + mobile number (e.g. Carrier +86138123456789): ",
		"topup_mobile_input_prompt_topup": "Please enter <b>{country}</b> carrier + mobile number (e.g. Carrier +86138123456789): ",
		"topup_mobile_saved_prompt_data":  "Tap a number below to top up, or re-enter <b>{country}</b> carrier + mobile number (e.g. Carrier +86138123456789): ",
		"topup_mobile_saved_prompt_topup": "Tap a number below to top up, or re-enter <b>{country}</b> carrier + mobile number (e.g. Carrier +86138123456789): ",
		"topup_mobile_manager_button":     "🔢 Manage Numbers",
		"topup_add_mobile_button":         "➕ Add Number",
		"topup_delete_mobile_button":      "➖ Delete Number",
		"topup_plan_prompt_data":          "Please select a 5G data plan:",
		"topup_plan_prompt_topup":         "Please select a top-up amount\n✅ Auto settled at real-time exchange rate\n✅ After successful top-up, a {fee_rate} service fee will be charged (deducted from the credited amount)",
	}
}

func cloneValues(src url.Values) url.Values {
	dst := make(url.Values, len(src))
	for key, values := range src {
		dst[key] = append([]string(nil), values...)
	}
	return dst
}

func writeJSON(w http.ResponseWriter, payload any) {
	_ = json.NewEncoder(w).Encode(payload)
}
