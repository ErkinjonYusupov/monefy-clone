package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	remindBefore   = 3 * 24 * time.Hour
	expiredRecent  = 7 * 24 * time.Hour
	reminderPeriod = time.Hour
)

// RunReminders soatiga bir marta premium muddatini tekshiradi va foydalanuvchiga
// (tugashiga 3 kun qolganda va tugaganda) hamda adminga xabar yuboradi.
func (b *Bot) RunReminders(ctx context.Context) {
	t := time.NewTicker(reminderPeriod)
	defer t.Stop()
	for {
		b.SendReminders(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (b *Bot) SendReminders(ctx context.Context) {
	soon, err := b.store.ClaimExpiringSoon(ctx, remindBefore)
	if err != nil {
		slog.Warn("reminders: expiring soon", "err", err)
	}
	for _, u := range soon {
		days := int(time.Until(*u.PremiumUntil).Hours()/24) + 1
		b.sendWithApp(ctx, u.TelegramID, fmt.Sprintf(
			"⏳ Premium obunangiz %d kundan keyin (%s) tugaydi.\nUzaytirish uchun ilovada «Premium» bo'limini oching.",
			days, u.PremiumUntil.Format("2006-01-02")), "Uzaytirish")
	}

	expired, err := b.store.ClaimExpired(ctx, expiredRecent)
	if err != nil {
		slog.Warn("reminders: expired", "err", err)
	}
	for _, u := range expired {
		b.sendWithApp(ctx, u.TelegramID,
			"⌛ Premium obunangiz tugadi. Ma'lumotlaringiz saqlanib qoldi.\nQayta faollashtirish uchun ilovada «Premium» bo'limini oching.",
			"Premium olish")
		b.NotifyAdmin(ctx, "⌛ Premium tugadi: "+UserLabel(&u))
	}
}

// sendWithApp xabarni Mini App'ni ochuvchi tugma bilan yuboradi. URL'ga "#/premium" qo'shilmaydi:
// Telegram ishga tushirish parametrlarini URL hash'iga yozadi va u marshrut bilan to'qnashardi.
func (b *Bot) sendWithApp(ctx context.Context, chatID int64, text, button string) {
	var kb *InlineKeyboard
	if b.webAppURL != "" {
		kb = &InlineKeyboard{InlineKeyboard: [][]InlineButton{{{Text: button, WebApp: &WebApp{URL: b.webAppURL}}}}}
	}
	if err := b.api.SendMessage(ctx, chatID, text, kb); err != nil {
		slog.Warn("reminder send", "chat", chatID, "err", err)
	}
}
