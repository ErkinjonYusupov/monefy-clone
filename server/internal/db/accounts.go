package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// accountSelect har bir hisobning joriy qoldig'ini (hisob valyutasida) ham hisoblaydi.
const accountSelect = `
	SELECT a.id, a.name, a.currency, a.initial_balance, a.icon, a.is_archived,
		a.initial_balance
		+ COALESCE((SELECT sum(CASE c.kind WHEN 'income' THEN t.amount ELSE -t.amount END)
			FROM transactions t JOIN categories c ON c.id = t.category_id
			WHERE t.account_id = a.id), 0)
		+ COALESCE((SELECT sum(amount_to) FROM transfers WHERE to_account_id = a.id), 0)
		- COALESCE((SELECT sum(amount_from) FROM transfers WHERE from_account_id = a.id), 0)
	FROM accounts a`

func scanAccount(row pgx.Row) (*Account, error) {
	var a Account
	if err := row.Scan(&a.ID, &a.Name, &a.Currency, &a.InitialBalance, &a.Icon, &a.IsArchived, &a.Balance); err != nil {
		return nil, notFound(err)
	}
	return &a, nil
}

func (s *Store) ListAccounts(ctx context.Context, userID int64) ([]Account, error) {
	rows, err := s.pool.Query(ctx, accountSelect+` WHERE a.user_id=$1 ORDER BY a.is_archived, a.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Account{}
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *Store) GetAccount(ctx context.Context, userID, id int64) (*Account, error) {
	return scanAccount(s.pool.QueryRow(ctx, accountSelect+` WHERE a.user_id=$1 AND a.id=$2`, userID, id))
}

type AccountInput struct {
	Name           *string `json:"name"`
	Currency       *string `json:"currency"`
	InitialBalance *int64  `json:"initial_balance"`
	Icon           *string `json:"icon"`
	IsArchived     *bool   `json:"is_archived"`
}

func (s *Store) CreateAccount(ctx context.Context, userID int64, in AccountInput) (*Account, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO accounts (user_id, name, currency, initial_balance, icon)
		VALUES ($1, $2, $3, COALESCE($4, 0), COALESCE($5, 'wallet')) RETURNING id`,
		userID, in.Name, in.Currency, in.InitialBalance, in.Icon).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetAccount(ctx, userID, id)
}

func (s *Store) UpdateAccount(ctx context.Context, userID, id int64, in AccountInput) (*Account, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE accounts SET
			name = COALESCE($3, name),
			currency = COALESCE($4, currency),
			initial_balance = COALESCE($5, initial_balance),
			icon = COALESCE($6, icon),
			is_archived = COALESCE($7, is_archived)
		WHERE user_id=$1 AND id=$2`,
		userID, id, in.Name, in.Currency, in.InitialBalance, in.Icon, in.IsArchived)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetAccount(ctx, userID, id)
}

// DeleteAccount hisobni va unga tegishli barcha yozuvlarni o'chiradi (CASCADE).
func (s *Store) DeleteAccount(ctx context.Context, userID, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM accounts WHERE user_id=$1 AND id=$2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) CountAccounts(ctx context.Context, userID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM accounts WHERE user_id=$1`, userID).Scan(&n)
	return n, err
}

// RenameBaseCurrency bepul tarifda asosiy valyuta o'zgarganda shu valyutadagi hisoblarni ham o'zgartiradi.
// Boshqa valyutadagi hisoblarga (premium davrida ochilgan) tegilmaydi: aks holda 100 USD 100 RUB bo'lib qolardi.
func (s *Store) RenameBaseCurrency(ctx context.Context, userID int64, from, to string) error {
	_, err := s.pool.Exec(ctx, `UPDATE accounts SET currency=$3 WHERE user_id=$1 AND currency=$2`, userID, from, to)
	return err
}
