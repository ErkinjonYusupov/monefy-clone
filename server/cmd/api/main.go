package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"monefy/internal/bot"
	"monefy/internal/config"
	"monefy/internal/db"
	"monefy/internal/handler"
	"monefy/migrations"
)

func main() {
	// .env repo ildizida; server/ ichidan ishga tushirilsa ../.env o'qiladi. Prod'da env Dokploy'dan keladi.
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// staticDir React build papkasi. Docker image'da u binary yonida (/app/web) turadi;
// lokalda bu papka yo'q, sahifani Vite dev server beradi.
func staticDir() string {
	if st, err := os.Stat("web"); err == nil && st.IsDir() {
		return "web"
	}
	return ""
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := db.EnsureDatabase(ctx, cfg.DatabaseURL); err != nil {
		return err
	}
	store, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.Migrate(ctx, migrations.FS); err != nil {
		return err
	}

	b := bot.New(bot.NewClient(cfg.BotToken), store, cfg.AdminChatID, cfg.WebAppURL)
	if os.Getenv("BOT_POLLING") != "false" {
		go b.Run(ctx)
	}
	go b.RunReminders(ctx)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler.New(cfg, store, b).Routes(staticDir()),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()

	slog.Info("server started", "port", cfg.Port, "dev_mode", cfg.DevTelegramID != 0)
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
