package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"monefy/internal/db"
	"monefy/internal/service"
)

type Bot struct {
	api         *Client
	store       *db.Store
	adminChatID int64
	webAppURL   string
}

func New(api *Client, store *db.Store, adminChatID int64, webAppURL string) *Bot {
	return &Bot{api: api, store: store, adminChatID: adminChatID, webAppURL: webAppURL}
}

func (b *Bot) API() *Client { return b.api }

// Run long polling orqali xabarlarni qabul qiladi, ctx bekor bo'lguncha ishlaydi.
func (b *Bot) Run(ctx context.Context) {
	var offset int64
	for ctx.Err() == nil {
		updates, err := b.api.GetUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("bot getUpdates", "err", err)
				time.Sleep(3 * time.Second)
			}
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			if u.Message != nil {
				b.handle(ctx, u.Message)
			}
		}
	}
}

func (b *Bot) reply(ctx context.Context, chatID int64, text string) {
	if err := b.api.SendMessage(ctx, chatID, text, nil); err != nil {
		slog.Warn("bot reply", "err", err)
	}
}

func (b *Bot) handle(ctx context.Context, m *Message) {
	if m.Contact != nil {
		b.contact(ctx, m)
		return
	}
	fields := strings.Fields(m.Text)
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return
	}
	// "/grant@MyBot" -> "/grant"
	cmd, _, _ := strings.Cut(fields[0], "@")
	args := fields[1:]

	if cmd == "/start" {
		b.start(ctx, m.Chat.ID)
		return
	}
	if b.adminChatID == 0 || m.Chat.ID != b.adminChatID {
		return
	}
	var adminID int64
	if m.From != nil {
		adminID = m.From.ID
	}
	switch cmd {
	case "/help":
		b.reply(ctx, m.Chat.ID, adminHelp)
	case "/stats":
		b.stats(ctx, m.Chat.ID)
	case "/user":
		b.userInfo(ctx, m.Chat.ID, args)
	case "/grant":
		b.grant(ctx, m.Chat.ID, adminID, args)
	case "/revoke":
		b.revoke(ctx, m.Chat.ID, args)
	}
}

const adminHelp = `<b>Admin buyruqlari</b>
/stats — foydalanuvchilar va tushum
/user &lt;telegram_id&gt; — foydalanuvchi va obunalari
/grant &lt;telegram_id&gt; &lt;week|month|3month|6month|year|lifetime|30d&gt; [summa] — premium berish
/revoke &lt;telegram_id&gt; — premiumni bekor qilish

Summa so'mda, ko'rsatilmasa reja narxi yoziladi (30d uchun 0).`

func (b *Bot) start(ctx context.Context, chatID int64) {
	text := "Monefy — xarajat va daromadlaringizni oson hisoblang."
	var kb *InlineKeyboard
	if b.webAppURL != "" {
		kb = &InlineKeyboard{InlineKeyboard: [][]InlineButton{{{Text: "Ilovani ochish", WebApp: &WebApp{URL: b.webAppURL}}}}}
	}
	if err := b.api.SendMessage(ctx, chatID, text, kb); err != nil {
		slog.Warn("bot start", "err", err)
	}
}

func (b *Bot) stats(ctx context.Context, chatID int64) {
	us, err := b.store.UserStats(ctx)
	if err != nil {
		b.reply(ctx, chatID, "Xato: "+html.EscapeString(err.Error()))
		return
	}
	rs, err := b.store.RevenueStats(ctx)
	if err != nil {
		b.reply(ctx, chatID, "Xato: "+html.EscapeString(err.Error()))
		return
	}
	b.reply(ctx, chatID, fmt.Sprintf(`<b>Statistika</b>
Foydalanuvchilar: %d (bugun +%d, 7 kunda faol: %d)
Premium: %d

<b>Obunalar</b>
Shu oy: %d ta, %s so'm
Jami: %d ta, %s so'm`,
		us.Total, us.NewToday, us.ActiveWeek, us.Premium,
		rs.MonthCount, service.FormatMoney(rs.MonthAmount), rs.TotalCount, service.FormatMoney(rs.TotalAmount)))
}

func (b *Bot) lookup(ctx context.Context, chatID int64, args []string) *db.User {
	if len(args) < 1 {
		b.reply(ctx, chatID, "telegram_id kerak. /help")
		return nil
	}
	tgID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		b.reply(ctx, chatID, "telegram_id son bo'lishi kerak")
		return nil
	}
	u, err := b.store.UserByTelegramID(ctx, tgID)
	if errors.Is(err, db.ErrNotFound) {
		b.reply(ctx, chatID, "Foydalanuvchi topilmadi (u ilovani hali ochmagan)")
		return nil
	}
	if err != nil {
		b.reply(ctx, chatID, "Xato: "+html.EscapeString(err.Error()))
		return nil
	}
	return u
}

func (b *Bot) userInfo(ctx context.Context, chatID int64, args []string) {
	u := b.lookup(ctx, chatID, args)
	if u == nil {
		return
	}
	subs, err := b.store.ListSubscriptions(ctx, u.ID)
	if err != nil {
		b.reply(ctx, chatID, "Xato: "+html.EscapeString(err.Error()))
		return
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>%s</b>\n", UserLabel(u))
	fmt.Fprintf(&sb, "Telefon: %s\n", phoneText(u.Phone))
	fmt.Fprintf(&sb, "Ro'yxatdan o'tgan: %s\n", u.CreatedAt.Format("2006-01-02"))
	fmt.Fprintf(&sb, "Premium: %s\n", PremiumStatus(u))
	if len(subs) > 0 {
		sb.WriteString("\n<b>Obunalar</b>\n")
		for _, s := range subs {
			status := ""
			if s.RevokedAt != nil {
				status = " (bekor qilingan)"
			}
			fmt.Fprintf(&sb, "#%d %s, %s, %s so'm, %s → %s%s\n", s.ID, s.Plan, s.Source,
				service.FormatMoney(s.Amount), s.StartsAt.Format("2006-01-02"), s.ExpiresAt.Format("2006-01-02"), status)
		}
	}
	b.reply(ctx, chatID, sb.String())
}

func (b *Bot) grant(ctx context.Context, chatID, adminID int64, args []string) {
	if len(args) < 2 {
		b.reply(ctx, chatID, "Foydalanish: /grant &lt;telegram_id&gt; &lt;month|year|lifetime|30d&gt; [summa]")
		return
	}
	plan, err := service.ParsePlan(args[1])
	if err != nil {
		b.reply(ctx, chatID, html.EscapeString(err.Error()))
		return
	}
	amount := plan.Price
	if len(args) >= 3 {
		sum, err := strconv.ParseInt(strings.ReplaceAll(args[2], "_", ""), 10, 64)
		if err != nil || sum < 0 {
			b.reply(ctx, chatID, "Summa butun son bo'lishi kerak (so'mda)")
			return
		}
		amount = sum * 100
	}
	u := b.lookup(ctx, chatID, args)
	if u == nil {
		return
	}
	sub, err := b.store.GrantSubscription(ctx, db.GrantInput{
		UserID: u.ID, Plan: plan.Code, Days: plan.Days, Source: "admin",
		Amount: amount, Currency: "UZS", GrantedBy: &adminID,
	})
	if err != nil {
		b.reply(ctx, chatID, "Xato: "+html.EscapeString(err.Error()))
		return
	}
	b.reply(ctx, chatID, fmt.Sprintf("✅ %s ga premium berildi (%s), %s gacha. Summa: %s so'm",
		UserLabel(u), plan.Title, untilText(sub.ExpiresAt), service.FormatMoney(amount)))
	b.NotifyPremium(ctx, u, sub)
}

func (b *Bot) revoke(ctx context.Context, chatID int64, args []string) {
	u := b.lookup(ctx, chatID, args)
	if u == nil {
		return
	}
	if err := b.store.RevokePremium(ctx, u.ID); err != nil {
		b.reply(ctx, chatID, "Xato: "+html.EscapeString(err.Error()))
		return
	}
	b.reply(ctx, chatID, fmt.Sprintf("Premium bekor qilindi: %s", UserLabel(u)))
}

// contact Mini App'dagi requestContact() natijasi botga ham kontakt xabari bo'lib keladi.
// Faqat foydalanuvchi o'z raqamini yuborgan bo'lsa saqlanadi.
func (b *Bot) contact(ctx context.Context, m *Message) {
	if m.From == nil || m.Contact.UserID != m.From.ID {
		b.reply(ctx, m.Chat.ID, "Iltimos, o'zingizning telefon raqamingizni yuboring.")
		return
	}
	if _, err := b.store.SetPhone(ctx, m.From.ID, service.NormalizePhone(m.Contact.PhoneNumber)); err != nil {
		if !errors.Is(err, db.ErrNotFound) {
			slog.Warn("bot save phone", "err", err)
		}
		return
	}
	b.reply(ctx, m.Chat.ID, "✅ Raqamingiz tasdiqlandi. Ilovaga qaytishingiz mumkin.")
}

// NotifyPremium foydalanuvchiga premium berilgani haqida xabar yuboradi.
func (b *Bot) NotifyPremium(ctx context.Context, u *db.User, sub *db.Subscription) {
	b.reply(ctx, u.TelegramID, fmt.Sprintf("🎉 Premium faollashtirildi: %s gacha. Rahmat!", untilText(sub.ExpiresAt)))
}

// NotifyAdmin admin chatiga xabar yuboradi (ADMIN_CHAT_ID bo'lmasa hech narsa qilmaydi).
func (b *Bot) NotifyAdmin(ctx context.Context, text string) {
	if b.adminChatID == 0 {
		return
	}
	if err := b.api.SendMessage(ctx, b.adminChatID, text, nil); err != nil {
		slog.Warn("notify admin", "err", err)
	}
}

func UserLabel(u *db.User) string {
	name := strings.TrimSpace(u.FirstName + " " + u.LastName)
	label := html.EscapeString(name)
	if u.Username != "" {
		label += " @" + html.EscapeString(u.Username)
	}
	return fmt.Sprintf("%s (<code>%d</code>)", label, u.TelegramID)
}

func phoneText(p string) string {
	if p == "" {
		return "yo'q"
	}
	return p
}

func PremiumStatus(u *db.User) string {
	if !u.IsPremium() {
		return "yo'q"
	}
	return untilText(*u.PremiumUntil) + " gacha"
}

func untilText(t time.Time) string {
	if time.Until(t) > 50*365*24*time.Hour {
		return "umrbod"
	}
	return t.Format("2006-01-02")
}
