package db

import "context"

const transferSelect = `
	SELECT id, from_account_id, to_account_id, amount_from, amount_to, note,
		to_char(occurred_on, 'YYYY-MM-DD'), created_at
	FROM transfers`

func (s *Store) ListTransfers(ctx context.Context, userID int64, p Period) ([]Transfer, error) {
	rows, err := s.pool.Query(ctx, transferSelect+`
		WHERE user_id=$1
			AND ($2 = '' OR occurred_on >= $2::date)
			AND ($3 = '' OR occurred_on <= $3::date)
		ORDER BY occurred_on DESC, id DESC`, userID, p.From, p.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transfer{}
	for rows.Next() {
		var t Transfer
		if err := rows.Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.AmountFrom, &t.AmountTo, &t.Note,
			&t.OccurredOn, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

type TransferInput struct {
	FromAccountID int64   `json:"from_account_id"`
	ToAccountID   int64   `json:"to_account_id"`
	AmountFrom    int64   `json:"amount_from"`
	AmountTo      int64   `json:"amount_to"` // 0 bo'lsa amount_from bilan teng (bir xil valyuta)
	Note          string  `json:"note"`
	OccurredOn    *string `json:"occurred_on"`
}

func (s *Store) CreateTransfer(ctx context.Context, userID int64, in TransferInput) (*Transfer, error) {
	var t Transfer
	err := s.pool.QueryRow(ctx, `
		INSERT INTO transfers (user_id, from_account_id, to_account_id, amount_from, amount_to, note, occurred_on)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7::date, current_date))
		RETURNING id, from_account_id, to_account_id, amount_from, amount_to, note,
			to_char(occurred_on, 'YYYY-MM-DD'), created_at`,
		userID, in.FromAccountID, in.ToAccountID, in.AmountFrom, in.AmountTo, in.Note, in.OccurredOn).
		Scan(&t.ID, &t.FromAccountID, &t.ToAccountID, &t.AmountFrom, &t.AmountTo, &t.Note, &t.OccurredOn, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) DeleteTransfer(ctx context.Context, userID, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM transfers WHERE user_id=$1 AND id=$2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
