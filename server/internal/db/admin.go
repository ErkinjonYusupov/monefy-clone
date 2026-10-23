package db

import (
	"context"
	"regexp"
	"strings"
)

var phoneLike = regexp.MustCompile(`^[+\d\s()-]*\d[+\d\s()-]*$`)

// SetPhone Telegram tasdiqlagan telefon raqamini saqlaydi.
func (s *Store) SetPhone(ctx context.Context, telegramID int64, phone string) (*User, error) {
	return scanUser(s.pool.QueryRow(ctx, `
		UPDATE users SET phone=$2, phone_verified_at=now() WHERE telegram_id=$1 RETURNING `+userCols,
		telegramID, phone))
}

func (s *Store) UserByID(ctx context.Context, id int64) (*User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
}

// SearchUsers admin ro'yxati: ism, familiya, username, telefon yoki telegram_id bo'yicha qidiradi.
func (s *Store) SearchUsers(ctx context.Context, query string, limit, offset int) ([]User, int, error) {
	q := strings.TrimPrefix(strings.TrimSpace(query), "@")
	// LIKE maxsus belgilarini oddiy belgi qilib qidirish
	like := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q) + "%"
	// Telefon/telegram_id bo'yicha faqat so'rov raqamga o'xshasa qidiriladi ("+998 90-123", "(90) 123"),
	// aks holda "test4" dagi "4" har bir raqamga mos kelib qolardi.
	digits := ""
	if phoneLike.MatchString(q) {
		digits = strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, q)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT `+userCols+`, count(*) OVER ()
		FROM users
		WHERE $1 = ''
			OR first_name ILIKE $2 OR last_name ILIKE $2 OR username ILIKE $2
			OR (first_name || ' ' || last_name) ILIKE $2
			OR ($3 <> '' AND (phone LIKE '%' || $3 || '%' OR telegram_id::text = $3))
		ORDER BY created_at DESC, id DESC
		LIMIT $4 OFFSET $5`, q, like, digits, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []User{}
	total := 0
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.LastName, &u.Username, &u.LanguageCode,
			&u.BaseCurrency, &u.BudgetAmount, &u.PremiumUntil, &u.CreatedAt, &u.LastSeenAt, &u.Phone, &total); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// Sahifa bo'sh bo'lsa (masalan oxiridan keyingi sahifa), umumiy sonni alohida olamiz
	if len(out) == 0 && offset > 0 {
		_, total, err = s.SearchUsers(ctx, query, 1, 0)
	}
	return out, total, err
}
