package db

import "context"

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

// Summary davr bo'yicha kategoriyalar kesimidagi yig'indini va davr oxiridagi umumiy qoldiqni
// asosiy valyutada qaytaradi. accountID = 0 bo'lsa barcha hisoblar.
func (s *Store) Summary(ctx context.Context, user *User, p Period, accountID int64) (*Summary, error) {
	sum := &Summary{Currency: user.BaseCurrency, Categories: []CategoryTotal{}, MissingRates: []string{}}

	rows, err := s.pool.Query(ctx, `
		SELECT t.category_id, c.kind,
			sum(CASE WHEN a.currency = u.base_currency THEN t.amount
				ELSE round(t.amount * COALESCE(r.rate, 1)) END)::bigint,
			count(*)
		FROM transactions t
		JOIN users u ON u.id = t.user_id
		JOIN categories c ON c.id = t.category_id
		JOIN accounts a ON a.id = t.account_id
		LEFT JOIN currency_rates r ON r.user_id = t.user_id AND r.currency = a.currency
		WHERE t.user_id = $1
			AND ($2 = '' OR t.occurred_on >= $2::date)
			AND ($3 = '' OR t.occurred_on <= $3::date)
			AND ($4 = 0 OR t.account_id = $4)
		GROUP BY t.category_id, c.kind
		ORDER BY 3 DESC`, user.ID, p.From, p.To, accountID)
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

	err = s.pool.QueryRow(ctx, `
		WITH bal AS (
			SELECT a.id, a.currency, a.initial_balance
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
		SELECT COALESCE(sum(CASE WHEN b.currency = u.base_currency THEN b.amount
			ELSE round(b.amount * COALESCE(r.rate, 1)) END), 0)::bigint
		FROM bal b
		JOIN users u ON u.id = $1
		LEFT JOIN currency_rates r ON r.user_id = $1 AND r.currency = b.currency`,
		user.ID, p.To, accountID).Scan(&sum.Balance)
	if err != nil {
		return nil, err
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
