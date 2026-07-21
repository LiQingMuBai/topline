package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultTranslationDir = "translations"
	defaultLang           = "zh"
)

// Config 统一管理应用运行期配置，避免业务代码直接散落读取环境变量。
type Config struct {
	MySQLDSN       string
	BotToken       string
	BotName        string
	AgentName      string
	NotifyChatID   int64
	BotDebug       bool
	TranslationDir string
	DefaultLang    string
	SupportedLangs []string
	OrderImagePath string
}

// Load 从 .env 读取所有配置。
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("加载 .env 失败: %w", err)
	}

	cfg := &Config{
		MySQLDSN:       strings.TrimSpace(os.Getenv("MYSQL_DSN")),
		BotToken:       strings.TrimSpace(os.Getenv("TG_BOT_API")),
		BotName:        strings.TrimSpace(os.Getenv("BOT_NAME")),
		AgentName:      strings.TrimSpace(os.Getenv("AGENT")),
		BotDebug:       getEnvBool("TG_DEBUG", false),
		NotifyChatID:   getEnvInt64("TOPUP_NOTIFY_CHAT_ID", 0),
		TranslationDir: getEnvOrDefault("TRANSLATIONS_DIR", defaultTranslationDir),
		DefaultLang:    getEnvOrDefault("DEFAULT_LANG", defaultLang),
		OrderImagePath: getEnvOrDefault("ORDER_IMAGE_PATH", "./static/CCTV.png"),
	}
	cfg.SupportedLangs = []string{cfg.DefaultLang}

	if cfg.MySQLDSN == "" {
		return nil, fmt.Errorf("缺少 MYSQL_DSN 配置")
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("缺少 TG_BOT_API 配置")
	}
	if cfg.AgentName == "" {
		return nil, fmt.Errorf("缺少 AGENT 配置")
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
