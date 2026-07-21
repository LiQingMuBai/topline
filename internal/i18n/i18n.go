package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ushield_bot/internal/config"
	"ushield_bot/internal/global"
)

// Load 将翻译文件装载到全局内存，兼容现有 service 层对 global.Translations 的使用方式。
func Load(cfg *config.Config) error {
	global.Mutex.Lock()
	defer global.Mutex.Unlock()

	global.TranslationsDir = cfg.TranslationDir
	global.DefaultLang = cfg.DefaultLang
	global.SupportedLangs = cfg.SupportedLangs
	global.Translations = make(map[string]map[string]string, len(cfg.SupportedLangs))

	for _, lang := range cfg.SupportedLangs {
		filePath := filepath.Join(cfg.TranslationDir, lang+".json")
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("加载翻译文件失败 %s: %w", filePath, err)
		}

		langTranslations := make(map[string]string)
		if err := json.Unmarshal(data, &langTranslations); err != nil {
			return fmt.Errorf("解析翻译文件失败 %s: %w", filePath, err)
		}

		global.Translations[lang] = langTranslations
	}

	if _, ok := global.Translations[cfg.DefaultLang]; !ok {
		return fmt.Errorf("默认语言不存在: %s", cfg.DefaultLang)
	}

	return nil
}

// T 获取翻译文本，不存在时回退到默认语言或原始 key。
func T(lang, key string) string {
	global.Mutex.RLock()
	defer global.Mutex.RUnlock()

	if langMap, ok := global.Translations[lang]; ok {
		if value, exists := langMap[key]; exists {
			return value
		}
	}

	if lang != global.DefaultLang {
		if value, exists := global.Translations[global.DefaultLang][key]; exists {
			return value
		}
	}

	return key
}

// TParam 基于翻译模板做参数替换。
func TParam(lang, key string, params map[string]string) string {
	text := T(lang, key)
	for name, value := range params {
		text = strings.ReplaceAll(text, "{"+name+"}", value)
	}
	return text
}
