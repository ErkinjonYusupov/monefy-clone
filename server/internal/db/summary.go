package db

import (
	"context"
	"sort"
)

func (s *Store) ListRates(ctx context.Context, userID int64) ([]Rate, error) {
	rows, err := s.pool.Query(ctx, `SELECT currency, rate::text, updated_at FROM currency_rates WHERE user_id=$1 ORDER BY currency`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Rate{}
	for rows.Next() {
		var r Rate
		if err := rows.Scan(&r.Currency, &r.Rate, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) UpsertRate(ctx context.Context, userID int64, currency, rate string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO currency_rates (user_id, currency, rate) VALUES ($1, $2, $3::numeric)
		ON CONFLICT (user_id, currency) DO UPDATE SET rate = EXCLUDED.rate, updated_at = now()`,
		userID, currency, rate)
	return err
}

func (s *Store) DeleteRate(ctx context.Context, userID int64, currency string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM currency_rates WHERE user_id=$1 AND currency=$2`, userID, currency)
	return err
}

// Summary davr bo'yicha kategoriyalar kesimidagi yig'indini va davr oxiridagi qoldiqni qaytaradi.
// accountID = 0 bo'lsa barcha hisoblar: umumiy sonlar asosiy valyutaga kurs bo'yicha o'giriladi
// (diagramma uchun), ByCurrency esa har bir valyutani o'girmasdan alohida beradi.
// Bitta hisob tanlansa, hamma summalar shu hisob valyutasida (kurssiz).
func (s *Store) Summary(ctx context.Context, user *User, p Period, accountID int64) (*Summary, error) {
	target := user.BaseCurrency
	if accountID != 0 {
		acc, err := s.GetAccount(ctx, user.ID, accountID)
		if err != nil {
			return nil, err
		}
		target = acc.Currency
	}
	sum := &Summary{Currency: target, Categories: []CategoryTotal{}, MissingRates: []string{}, ByCurrency: []CurrencyTotal{}}

	// Kategoriyalar bo'yicha: target valyutaga o'girilgan
	rows, err := s.pool.Query(ctx, `
		SELECT t.category_id, c.kind,
			sum(CASE WHEN a.currency = $5 THEN t.amount
				ELSE round(t.amount * COALESCE(r.rate, 1)) END)::bigint,
			count(*)
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		JOIN accounts a ON a.id = t.account_id
		LEFT JOIN currency_rates r ON r.user_id = t.user_id AND r.currency = a.currency
		WHERE t.user_id = $1
			AND ($2 = '' OR t.occurred_on >= $2::date)
			AND ($3 = '' OR t.occurred_on <= $3::date)
			AND ($4 = 0 OR t.account_id = $4)
		GROUP BY t.category_id, c.kind
		ORDER BY 3 DESC`, user.ID, p.From, p.To, accountID, target)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var ct CategoryTotal
		if err := rows.Scan(&ct.CategoryID, &ct.Kind, &ct.Total, &ct.Count); err != nil {
			rows.Close()
			return nil, err
		}
		if ct.Kind == "income" {
			sum.Income += ct.Total
		} else {
			sum.Expense += ct.Total
		}
		sum.Categories = append(sum.Categories, ct)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	byCur := map[string]*CurrencyTotal{}
	get := func(c string) *CurrencyTotal {
		if byCur[c] == nil {
			byCur[c] = &CurrencyTotal{Currency: c}
		}
		return byCur[c]
	}

	// Valyutalar bo'yicha daromad/xarajat: o'girilmagan
	rows, err = s.pool.Query(ctx, `
		SELECT a.currency,
			COALESCE(sum(t.amount) FILTER (WHERE c.kind = 'income'), 0)::bigint,
			COALESCE(sum(t.amount) FILTER (WHERE c.kind = 'expense'), 0)::bigint
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		JOIN accounts a ON a.id = t.account_id
		WHERE t.user_id = $1
			AND ($2 = '' OR t.occurred_on >= $2::date)
			AND ($3 = '' OR t.occurred_on <= $3::date)
			AND ($4 = 0 OR t.account_id = $4)
		GROUP BY a.currency`, user.ID, p.From, p.To, accountID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c string
		var inc, exp int64
		if err := rows.Scan(&c, &inc, &exp); err != nil {
			rows.Close()
			return nil, err
		}
		ct := get(c)
		ct.Income, ct.Expense = inc, exp
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Davr oxiridagi qoldiq: valyuta bo'yicha o'girilmagan va jami target valyutada
	rows, err = s.pool.Query(ctx, `
		WITH bal AS (
			SELECT a.currency, a.initial_balance
				+ COALESCE((SELECT sum(CASE c.kind WHEN 'income' THEN t.amount ELSE -t.amount END)
					FROM transactions t JOIN categories c ON c.id = t.category_id
					WHERE t.account_id = a.id AND ($2 = '' OR t.occurred_on <= $2::date)), 0)
				+ COALESCE((SELECT sum(amount_to) FROM transfers
					WHERE to_account_id = a.id AND ($2 = '' OR occurred_on <= $2::date)), 0)
				- COALESCE((SELECT sum(amount_from) FROM transfers
					WHERE from_account_id = a.id AND ($2 = '' OR occurred_on <= $2::date)), 0) AS amount
			FROM accounts a
			WHERE a.user_id = $1 AND ($3 = 0 OR a.id = $3)
		)
		SELECT b.currency, sum(b.amount)::bigint,
			sum(CASE WHEN b.currency = $4 THEN b.amount ELSE round(b.amount * COALESCE(r.rate, 1)) END)::bigint
		FROM bal b
		LEFT JOIN currency_rates r ON r.user_id = $1 AND r.currency = b.currency
		GROUP BY b.currency`, user.ID, p.To, accountID, target)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var c string
		var native, converted int64
		if err := rows.Scan(&c, &native, &converted); err != nil {
			rows.Close()
			return nil, err
		}
		get(c).Balance = native
		sum.Balance += converted
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Asosiy (target) valyuta birinchi, qolganlari alifbo bo'yicha
	for _, ct := range byCur {
		sum.ByCurrency = append(sum.ByCurrency, *ct)
	}
	sort.Slice(sum.ByCurrency, func(i, j int) bool {
		a, b := sum.ByCurrency[i].Currency, sum.ByCurrency[j].Currency
		if (a == target) != (b == target) {
			return a == target
		}
		return a < b
	})

	if accountID != 0 {
		return sum, nil
	}
	rows, err = s.pool.Query(ctx, `
		SELECT DISTINCT a.currency FROM accounts a
		LEFT JOIN currency_rates r ON r.user_id = a.user_id AND r.currency = a.currency
		WHERE a.user_id = $1 AND a.currency <> $2 AND r.rate IS NULL`, user.ID, user.BaseCurrency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		sum.MissingRates = append(sum.MissingRates, c)
	}
	return sum, rows.Err()
}
