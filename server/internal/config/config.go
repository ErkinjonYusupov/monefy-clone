package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        string
	DatabaseURL string
	BotToken    string
	AdminChatID int64
	WebAppURL   string
	// DevTelegramID bo'lsa, initData tekshirilmaydi (faqat lokal brauzerda test uchun).
	DevTelegramID int64
	CORSOrigins   []string
}

func Load() (*Config, error) {
	c := &Config{
		Port:        env("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		BotToken:    os.Getenv("BOT_TOKEN"),
		WebAppURL:   os.Getenv("WEBAPP_URL"),
	}
	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	var err error
	if v := os.Getenv("ADMIN_CHAT_ID"); v != "" {
		if c.AdminChatID, err = strconv.ParseInt(v, 10, 64); err != nil {
			return nil, fmt.Errorf("ADMIN_CHAT_ID: %w", err)
		}
	}
	if v := os.Getenv("DEV_TELEGRAM_ID"); v != "" {
		if c.DevTelegramID, err = strconv.ParseInt(v, 10, 64); err != nil {
			return nil, fmt.Errorf("DEV_TELEGRAM_ID: %w", err)
		}
	}
	for _, o := range strings.Split(env("CORS_ORIGINS", "http://localhost:5173"), ",") {
		if o = strings.TrimSpace(o); o != "" {
			c.CORSOrigins = append(c.CORSOrigins, o)
		}
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
