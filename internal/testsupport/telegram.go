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
		"insufficient_balance_tips":    "余额不足，请先充值",
		"user_id":                      "用户ID",
		"username":                     "用户名",
		"balance":                      "余额",
		"deposit":                      "充值",
		"order_id":                     "订单号",
		"successfully_purchased_order": "✅成功购买价值{amount}USDT的话费或流量，请耐心等待帮您充值，有任何问题，请联系客服 @PolyTopUp",
		"deposit_records":              "充值记录",
		"prev":                         "上一页",
		"next":                         "下一页",
		"back_home":                    "返回个人中心",
		"back_homepage":                "返回首页",
		"support":                      "客服",
		"menu_topup":                   "⛽话费充值",
		"menu_data":                    "🔋流量充值",
		"menu_profile":                 "👤个人中心",
		"menu_support":                 "🐍联系娘子",
		"menu_language":                "🌐切换语言",
		"welcome_tips":                 "欢迎使用中文菜单",
		"hide_keyboard_tips":           "键盘已隐藏，发送 /start 重新显示",
		"language_menu_title":          "请选择语言 / Please choose a language",
		"language_option_zh":           "中文",
		"language_option_en":           "English",
		"language_switched":            "语言已切换成功。",
		"support_message":              "📞{support}\n工作时间：{work_time}\n",
	}
	global.Translations["en"] = map[string]string{
		"insufficient_balance_tips":    "Insufficient balance. Please top up first.",
		"user_id":                      "User ID",
		"username":                     "Username",
		"balance":                      "Balance",
		"deposit":                      "Deposit",
		"order_id":                     "Order ID",
		"successfully_purchased_order": "Successfully purchased {amount} USDT worth of top-up or data.",
		"deposit_records":              "Deposit Records",
		"prev":                         "Previous",
		"next":                         "Next",
		"back_home":                    "Back to Profile",
		"back_homepage":                "Back to Home",
		"support":                      "Support",
		"menu_topup":                   "⛽ Top Up",
		"menu_data":                    "🔋 Data Top Up",
		"menu_profile":                 "👤 Profile",
		"menu_support":                 "🐍 Support",
		"menu_language":                "🌐 Language",
		"welcome_tips":                 "Welcome to the English menu",
		"hide_keyboard_tips":           "Keyboard hidden. Send /start to show it again.",
		"language_menu_title":          "Please choose a language",
		"language_option_zh":           "中文",
		"language_option_en":           "English",
		"language_switched":            "Language switched successfully.",
		"support_message":              "📞{support}\nWorking hours: {work_time}\n",
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
