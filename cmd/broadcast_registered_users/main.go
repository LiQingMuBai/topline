package main

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ushield_bot/internal/config"
	"ushield_bot/internal/infrastructure/repositories"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type options struct {
	DryRun  bool
	Limit   int
	SleepMS int
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	opts := parseFlags()

	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("加载配置失败")
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		logrus.WithError(err).Fatal("连接 MySQL 失败")
	}

	sqlDB, err := db.DB()
	if err == nil {
		defer func() {
			_ = sqlDB.Close()
		}()
	}

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		logrus.WithError(err).Fatal("初始化 Telegram Bot 失败")
	}
	bot.Debug = cfg.BotDebug

	users, err := repositories.NewUserRepository(db).ListRegisteredUsers(context.Background(), "")
	if err != nil {
		logrus.WithError(err).Fatal("查询已注册用户失败")
	}

	seenChatIDs := make(map[int64]struct{}, len(users))
	sentCount := 0
	skippedCount := 0
	failedCount := 0

	for _, user := range users {
		if opts.Limit > 0 && sentCount >= opts.Limit {
			break
		}

		chatID, parseOK := parseChatID(user.Associates)
		if !parseOK {
			skippedCount++
			logrus.WithField("associates", user.Associates).Warn("跳过无效 chat_id")
			continue
		}
		if _, exists := seenChatIDs[chatID]; exists {
			skippedCount++
			continue
		}
		seenChatIDs[chatID] = struct{}{}

		text := releaseMessage(user.Lang, cfg.SupportUsername)
		if opts.DryRun {
			sentCount++
			logrus.WithFields(logrus.Fields{
				"chat_id": chatID,
				"lang":    normalizeLang(user.Lang),
				"text":    text,
			}).Info("dry-run 预览通知")
			continue
		}

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "HTML"
		if _, err := bot.Send(msg); err != nil {
			failedCount++
			logrus.WithError(err).WithFields(logrus.Fields{
				"chat_id":    chatID,
				"username":   user.Username,
				"associates": user.Associates,
			}).Warn("发送新版上线通知失败")
			continue
		}

		sentCount++
		logrus.WithFields(logrus.Fields{
			"chat_id":  chatID,
			"username": user.Username,
		}).Info("发送新版上线通知成功")

		if opts.SleepMS > 0 {
			time.Sleep(time.Duration(opts.SleepMS) * time.Millisecond)
		}
	}

	logrus.WithFields(logrus.Fields{
		"bot_name": cfg.BotName,
		"dry_run":  opts.DryRun,
		"total":    len(users),
		"sent":     sentCount,
		"skipped":  skippedCount,
		"failed":   failedCount,
	}).Info("广播结束")
}

func parseFlags() options {
	dryRun := flag.Bool("dry-run", true, "仅预览发送对象，不真正发消息")
	limit := flag.Int("limit", 0, "限制最大发送人数，0 表示不限制")
	sleepMS := flag.Int("sleep-ms", 1000, "每次发送后的间隔毫秒数")
	flag.Parse()

	return options{
		DryRun:  *dryRun,
		Limit:   *limit,
		SleepMS: *sleepMS,
	}
}

func parseChatID(value string) (int64, bool) {
	chatID, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || chatID == 0 {
		return 0, false
	}
	return chatID, true
}

func releaseMessage(lang, supportUsername string) string {
	supportLine := ""
	if support := strings.TrimSpace(supportUsername); support != "" {
		supportLine = "\n\n如需话费充值、流量充值或推广咨询，请联系 " + support
		if normalizeLang(lang) == "en" {
			supportLine = "\n\nFor mobile top-up, data top-up, or promotion inquiries, please contact " + support
		}
	}

	if normalizeLang(lang) == "en" {
		return "📢 <b>The new version of the bot is now live</b>\n\nThe latest mobile top-up and data top-up features are now available.\nPlease send /start to refresh the latest menu.\nIf you do not see the new features, send /hide first and then /start.\n\nReferral and promotion cooperation is also available now." + supportLine
	}
	return "📢 <b>新版本 Bot 已经上线</b>\n\n最新的话费充值、流量充值功能已经开放使用。\n请发送 /start 刷新最新菜单。\n如果暂时没有看到新功能，请先发送 /hide，再发送 /start。\n\n如需推广合作或推广咨询，也可以直接联系我们。" + supportLine
}

func normalizeLang(lang string) string {
	if strings.EqualFold(strings.TrimSpace(lang), "en") {
		return "en"
	}
	return "zh"
}

func exampleCommand() string {
	return fmt.Sprintf("go run ./cmd/broadcast_registered_users -dry-run=false -limit=100 -sleep-ms=80")
}
