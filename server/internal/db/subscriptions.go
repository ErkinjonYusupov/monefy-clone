package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type GrantInput struct {
	UserID        int64
	Plan          string
	Days          int // 0 = umrbod
	Source        string
	Amount        int64
	Currency      string
	GrantedBy     *int64
	ProviderTxnID *string
	Note          string
}

const lifetime = 100 * 365 * 24 * time.Hour

// GrantSubscription premiumni beradi yoki uzaytiradi: yangi muddat joriy muddat
// tugaganidan (yoki hozirdan) boshlanadi.
func (s *Store) GrantSubscription(ctx context.Context, in GrantInput) (*Subscription, error) {
	var sub Subscription
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var current *time.Time
		if err := tx.QueryRow(ctx, `SELECT premium_until FROM users WHERE id=$1 FOR UPDATE`, in.UserID).
			Scan(&current); err != nil {
			return notFound(err)
		}
		start := time.Now()
		if current != nil && current.After(start) {
			start = *current
		}
		dur := lifetime
		if in.Days > 0 {
			dur = time.Duration(in.Days) * 24 * time.Hour
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO subscriptions (user_id, plan, source, amount, currency, provider_txn_id, granted_by, starts_at, expires_at, note)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id, user_id, plan, source, amount, currency, granted_by, starts_at, expires_at, revoked_at, note, created_at`,
			in.UserID, in.Plan, in.Source, in.Amount, in.Currency, in.ProviderTxnID, in.GrantedBy,
			start, start.Add(dur), in.Note).
			Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Source, &sub.Amount, &sub.Currency, &sub.GrantedBy,
				&sub.StartsAt, &sub.ExpiresAt, &sub.RevokedAt, &sub.Note, &sub.CreatedAt)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE users SET premium_until=$2 WHERE id=$1`, in.UserID, sub.ExpiresAt)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

// RevokePremium foydalanuvchining amaldagi barcha obunalarini bekor qiladi.
func (s *Store) RevokePremium(ctx context.Context, userID int64) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE subscriptions SET revoked_at=now()
			WHERE user_id=$1 AND revoked_at IS NULL AND expires_at > now()`, userID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE users SET premium_until=NULL WHERE id=$1`, userID)
		if err == nil && tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return err
	})
}

func (s *Store) ListSubscriptions(ctx context.Context, userID int64) ([]Subscription, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, plan, source, amount, currency, granted_by, starts_at, expires_at, revoked_at, note, created_at
		FROM subscriptions WHERE user_id=$1 ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Subscription{}
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.UserID, &sub.Plan, &sub.Source, &sub.Amount, &sub.Currency, &sub.GrantedBy,
			&sub.StartsAt, &sub.ExpiresAt, &sub.RevokedAt, &sub.Note, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

type RevenueStats struct {
	MonthCount  int   `json:"month_count"`
	MonthAmount int64 `json:"month_amount"`
	TotalCount  int   `json:"total_count"`
	TotalAmount int64 `json:"total_amount"`
}

// RevenueStats bekor qilinmagan obunalar bo'yicha (UZS) tushum.
func (s *Store) RevenueStats(ctx context.Context) (RevenueStats, error) {
	var r RevenueStats
	err := s.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE created_at >= date_trunc('month', now())),
		COALESCE(sum(amount) FILTER (WHERE created_at >= date_trunc('month', now())), 0)::bigint,
		count(*),
		COALESCE(sum(amount), 0)::bigint
		FROM subscriptions WHERE revoked_at IS NULL AND currency = 'UZS'`).
		Scan(&r.MonthCount, &r.MonthAmount, &r.TotalCount, &r.TotalAmount)
	return r, err
}
