package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

const userCols = `id, telegram_id, first_name, last_name, username, language_code,
	base_currency, budget_amount, premium_until, created_at, last_seen_at, phone`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	err := row.Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.LastName, &u.Username, &u.LanguageCode,
		&u.BaseCurrency, &u.BudgetAmount, &u.PremiumUntil, &u.CreatedAt, &u.LastSeenAt, &u.Phone)
	if err != nil {
		return nil, notFound(err)
	}
	return &u, nil
}

type UpsertUser struct {
	TelegramID   int64
	FirstName    string
	LastName     string
	Username     string
	LanguageCode string
}

// UpsertUser foydalanuvchini yaratadi yoki yangilaydi. Yangi foydalanuvchiga
// standart hisoblar va kategoriyalar qo'shiladi; created=true qaytadi.
func (s *Store) UpsertUser(ctx context.Context, in UpsertUser) (u *User, created bool, err error) {
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO users (telegram_id, first_name, last_name, username, language_code)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (telegram_id) DO UPDATE SET
				first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name,
				username = EXCLUDED.username, language_code = EXCLUDED.language_code,
				last_seen_at = now()
			RETURNING `+userCols+`, (xmax = 0)`,
			in.TelegramID, in.FirstName, in.LastName, in.Username, in.LanguageCode)
		var x User
		if err := row.Scan(&x.ID, &x.TelegramID, &x.FirstName, &x.LastName, &x.Username, &x.LanguageCode,
			&x.BaseCurrency, &x.BudgetAmount, &x.PremiumUntil, &x.CreatedAt, &x.LastSeenAt, &x.Phone, &created); err != nil {
			return err
		}
		u = &x
		if created {
			return seedDefaults(ctx, tx, x.ID, x.BaseCurrency)
		}
		return nil
	})
	return u, created, err
}

// lastSeenEvery last_seen_at qanchalik tez-tez yangilanadi (statistika uchun shu aniqlik yetarli).
const lastSeenEvery = 10 * time.Minute

// AuthUser har bir API so'rovida chaqiriladi. Odatda faqat o'qiydi; bazaga faqat yangi foydalanuvchi
// bo'lsa, Telegram profili o'zgargan bo'lsa yoki last_seen_at eskirgan bo'lsa yozadi.
func (s *Store) AuthUser(ctx context.Context, in UpsertUser) (u *User, created bool, err error) {
	u, err = s.UserByTelegramID(ctx, in.TelegramID)
	if errors.Is(err, ErrNotFound) {
		return s.UpsertUser(ctx, in)
	}
	if err != nil {
		return nil, false, err
	}
	changed := u.FirstName != in.FirstName || u.LastName != in.LastName ||
		u.Username != in.Username || u.LanguageCode != in.LanguageCode
	if changed || time.Since(u.LastSeenAt) > lastSeenEvery {
		return s.UpsertUser(ctx, in)
	}
	return u, false, nil
}

func (s *Store) UserByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE telegram_id=$1`, telegramID))
}

type UserSettings struct {
	BaseCurrency *string
	BudgetAmount *int64
	ClearBudget  bool
}

func (s *Store) UpdateSettings(ctx context.Context, userID int64, in UserSettings) (*User, error) {
	return scanUser(s.pool.QueryRow(ctx, `
		UPDATE users SET
			base_currency = COALESCE($2, base_currency),
			budget_amount = CASE WHEN $4 THEN NULL ELSE COALESCE($3, budget_amount) END
		WHERE id=$1 RETURNING `+userCols,
		userID, in.BaseCurrency, in.BudgetAmount, in.ClearBudget))
}

type UserStats struct {
	Total      int `json:"total"`
	Premium    int `json:"premium"`
	NewToday   int `json:"new_today"`
	ActiveWeek int `json:"active_week"`
}

func (s *Store) UserStats(ctx context.Context) (UserStats, error) {
	var st UserStats
	err := s.pool.QueryRow(ctx, `SELECT
		count(*),
		count(*) FILTER (WHERE premium_until > now()),
		count(*) FILTER (WHERE created_at >= date_trunc('day', now())),
		count(*) FILTER (WHERE last_seen_at >= now() - interval '7 days')
		FROM users`).Scan(&st.Total, &st.Premium, &st.NewToday, &st.ActiveWeek)
	return st, err
}

type defaultCategory struct {
	kind, name, icon, color string
}

var defaultCategories = []defaultCategory{
	{"expense", "Oziq-ovqat", "shopping-cart", "#4CAF50"},
	{"expense", "Kafe va restoran", "utensils", "#FF9800"},
	{"expense", "Transport", "bus", "#2196F3"},
	{"expense", "Taksi", "car-taxi-front", "#FFC107"},
	{"expense", "Avtomobil", "car", "#607D8B"},
	{"expense", "Uy", "house", "#795548"},
	{"expense", "Kommunal to'lovlar", "receipt", "#009688"},
	{"expense", "Aloqa", "smartphone", "#3F51B5"},
	{"expense", "Kiyim", "shirt", "#9C27B0"},
	{"expense", "Salomatlik", "heart-pulse", "#F44336"},
	{"expense", "Gigiyena", "bath", "#00BCD4"},
	{"expense", "Ko'ngilochar", "gamepad-2", "#E91E63"},
	{"expense", "Sport", "dumbbell", "#8BC34A"},
	{"expense", "Uy hayvonlari", "paw-print", "#A1887F"},
	{"expense", "Sovg'alar", "gift", "#FF5722"},
	{"income", "Maosh", "briefcase", "#43A047"},
	{"income", "Depozit", "landmark", "#1E88E5"},
	{"income", "Jamg'arma", "piggy-bank", "#FB8C00"},
}

func seedDefaults(ctx context.Context, tx pgx.Tx, userID int64, currency string) error {
	batch := &pgx.Batch{}
	batch.Queue(`INSERT INTO accounts (user_id, name, currency, icon) VALUES ($1, 'Naqd pul', $2, 'wallet'), ($1, 'Karta', $2, 'credit-card')`,
		userID, currency)
	for i, c := range defaultCategories {
		batch.Queue(`INSERT INTO categories (user_id, kind, name, icon, color, sort_order) VALUES ($1,$2,$3,$4,$5,$6)`,
			userID, c.kind, c.name, c.icon, c.color, i)
	}
	return tx.SendBatch(ctx, batch).Close()
}
