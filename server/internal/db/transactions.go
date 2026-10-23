package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const txSelect = `
	SELECT t.id, t.account_id, t.category_id, c.kind, t.amount, a.currency, t.note,
		to_char(t.occurred_on, 'YYYY-MM-DD'), t.created_at
	FROM transactions t
	JOIN categories c ON c.id = t.category_id
	JOIN accounts a ON a.id = t.account_id`

func scanTx(row pgx.Row) (*Transaction, error) {
	var t Transaction
	if err := row.Scan(&t.ID, &t.AccountID, &t.CategoryID, &t.Kind, &t.Amount, &t.Currency, &t.Note,
		&t.OccurredOn, &t.CreatedAt); err != nil {
		return nil, notFound(err)
	}
	return &t, nil
}

// Period [From, To] oralig'i, ikkalasi ham kiradi. Bo'sh qiymat = chegara yo'q.
type Period struct {
	From string // YYYY-MM-DD
	To   string
}

type TxFilter struct {
	Period
	AccountID  int64
	CategoryID int64
	Kind       string
	Limit      int
}

func (s *Store) ListTransactions(ctx context.Context, userID int64, f TxFilter) ([]Transaction, error) {
	where := []string{"t.user_id = $1"}
	args := []any{userID}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}
	if f.From != "" {
		add("t.occurred_on >= $%d", f.From)
	}
	if f.To != "" {
		add("t.occurred_on <= $%d", f.To)
	}
	if f.AccountID != 0 {
		add("t.account_id = $%d", f.AccountID)
	}
	if f.CategoryID != 0 {
		add("t.category_id = $%d", f.CategoryID)
	}
	if f.Kind != "" {
		add("c.kind = $%d", f.Kind)
	}
	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	q := txSelect + " WHERE " + strings.Join(where, " AND ") +
		fmt.Sprintf(" ORDER BY t.occurred_on DESC, t.id DESC LIMIT %d", limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transaction{}
	for rows.Next() {
		t, err := scanTx(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func (s *Store) GetTransaction(ctx context.Context, userID, id int64) (*Transaction, error) {
	return scanTx(s.pool.QueryRow(ctx, txSelect+` WHERE t.user_id=$1 AND t.id=$2`, userID, id))
}

type TxInput struct {
	AccountID  *int64  `json:"account_id"`
	CategoryID *int64  `json:"category_id"`
	Amount     *int64  `json:"amount"`
	Note       *string `json:"note"`
	OccurredOn *string `json:"occurred_on"`
}

func (s *Store) CreateTransaction(ctx context.Context, userID int64, in TxInput) (*Transaction, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO transactions (user_id, account_id, category_id, amount, note, occurred_on)
		VALUES ($1, $2, $3, $4, COALESCE($5, ''), COALESCE($6::date, current_date)) RETURNING id`,
		userID, in.AccountID, in.CategoryID, in.Amount, in.Note, in.OccurredOn).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetTransaction(ctx, userID, id)
}

func (s *Store) UpdateTransaction(ctx context.Context, userID, id int64, in TxInput) (*Transaction, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE transactions SET
			account_id = COALESCE($3, account_id),
			category_id = COALESCE($4, category_id),
			amount = COALESCE($5, amount),
			note = COALESCE($6, note),
			occurred_on = COALESCE($7::date, occurred_on)
		WHERE user_id=$1 AND id=$2`,
		userID, id, in.AccountID, in.CategoryID, in.Amount, in.Note, in.OccurredOn)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetTransaction(ctx, userID, id)
}

func (s *Store) DeleteTransaction(ctx context.Context, userID, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM transactions WHERE user_id=$1 AND id=$2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ExportRow CSV eksport uchun bitta qator.
type ExportRow struct {
	Date     string
	Kind     string
	Account  string
	Category string
	Amount   int64
	Currency string
	Note     string
}

func (s *Store) ExportTransactions(ctx context.Context, userID int64, p Period) ([]ExportRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(t.occurred_on, 'YYYY-MM-DD'), c.kind, a.name, c.name, t.amount, a.currency, t.note
		FROM transactions t
		JOIN categories c ON c.id = t.category_id
		JOIN accounts a ON a.id = t.account_id
		WHERE t.user_id = $1
			AND ($2 = '' OR t.occurred_on >= $2::date)
			AND ($3 = '' OR t.occurred_on <= $3::date)
		ORDER BY t.occurred_on, t.id`, userID, p.From, p.To)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExportRow
	for rows.Next() {
		var r ExportRow
		if err := rows.Scan(&r.Date, &r.Kind, &r.Account, &r.Category, &r.Amount, &r.Currency, &r.Note); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
