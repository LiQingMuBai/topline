package app

import (
	"fmt"

	"ushield_bot/internal/cache"
	"ushield_bot/internal/config"
	"ushield_bot/internal/domain"
	"ushield_bot/internal/i18n"
	"ushield_bot/internal/telegram"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// App 统一管理应用启动、依赖注入和生命周期。
type App struct {
	cfg    *config.Config
	db     *gorm.DB
	bot    *tgbotapi.BotAPI
	cache  cache.Cache
	router *telegram.Router
}

// New 初始化运行所需依赖。
func New() (*App, error) {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if err := i18n.Load(cfg); err != nil {
		return nil, err
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}
	if err := db.AutoMigrate(&domain.UserUSDTDeposits{}); err != nil {
		return nil, fmt.Errorf("同步订单表结构失败: %w", err)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("初始化 Telegram Bot 失败: %w", err)
	}
	bot.Debug = cfg.BotDebug

	appCache := cache.NewMemoryCache()
	router := telegram.NewRouter(cfg, db, bot, appCache)

	return &App{
		cfg:    cfg,
		db:     db,
		bot:    bot,
		cache:  appCache,
		router: router,
	}, nil
}

// Run 启动消息消费循环。
func (a *App) Run() error {
	if _, err := a.bot.Request(tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "start", Description: "start"},
		tgbotapi.BotCommand{Command: "hide", Description: "hide"},
	)); err != nil {
		logrus.WithError(err).Warn("设置机器人命令失败")
	}

	logrus.WithFields(logrus.Fields{
		"bot_name": a.bot.Self.UserName,
		"app_name": a.cfg.BotName,
	}).Info("Telegram 机器人已启动")

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updates := a.bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		a.router.HandleUpdate(update)
	}

	return nil
}

// Close 释放可关闭资源。
func (a *App) Close() error {
	_ = a.db
	return a.cache.Close()
}
