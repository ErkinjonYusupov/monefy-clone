package db

import (
	"context"
	"time"
)

// ClaimExpiringSoon muddati `within` ichida tugaydigan va hali eslatilmagan foydalanuvchilarni
// "eslatildi" deb belgilaydi va qaytaradi. Belgilash va tanlash bitta UPDATE'da: bir nechta server
// nusxasi ishlasa ham har bir eslatma faqat bir marta yuboriladi.
func (s *Store) ClaimExpiringSoon(ctx context.Context, within time.Duration) ([]User, error) {
	return s.claim(ctx, `
		UPDATE users SET premium_reminded_for = premium_until
		WHERE premium_until > now() AND premium_until <= now() + make_interval(secs => $1)
			AND premium_reminded_for IS DISTINCT FROM premium_until
		RETURNING `+userCols, within.Seconds())
}

// ClaimExpired yaqinda (`recent` ichida) tugagan va hali xabar berilmagan obunalarni belgilaydi.
// `recent` cheklovi birinchi deploy'da eski tugagan obunalar uchun xabar yog'ilib ketmasligi uchun.
func (s *Store) ClaimExpired(ctx context.Context, recent time.Duration) ([]User, error) {
	return s.claim(ctx, `
		UPDATE users SET premium_expired_for = premium_until
		WHERE premium_until <= now() AND premium_until > now() - make_interval(secs => $1)
			AND premium_expired_for IS DISTINCT FROM premium_until
		RETURNING `+userCols, recent.Seconds())
}

func (s *Store) claim(ctx context.Context, sql string, args ...any) ([]User, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}
