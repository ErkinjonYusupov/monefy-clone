package db

import (
	"context"
	"errors"
)

var ErrCategoryInUse = errors.New("category has transactions")

func (s *Store) ListCategories(ctx context.Context, userID int64) ([]Category, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, kind, name, icon, color, sort_order, is_archived
		FROM categories WHERE user_id=$1 ORDER BY kind, sort_order, id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Kind, &c.Name, &c.Icon, &c.Color, &c.SortOrder, &c.IsArchived); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetCategory(ctx context.Context, userID, id int64) (*Category, error) {
	var c Category
	err := s.pool.QueryRow(ctx, `
		SELECT id, kind, name, icon, color, sort_order, is_archived
		FROM categories WHERE user_id=$1 AND id=$2`, userID, id).
		Scan(&c.ID, &c.Kind, &c.Name, &c.Icon, &c.Color, &c.SortOrder, &c.IsArchived)
	if err != nil {
		return nil, notFound(err)
	}
	return &c, nil
}

type CategoryInput struct {
	Kind       *string `json:"kind"`
	Name       *string `json:"name"`
	Icon       *string `json:"icon"`
	Color      *string `json:"color"`
	SortOrder  *int    `json:"sort_order"`
	IsArchived *bool   `json:"is_archived"`
}

func (s *Store) CreateCategory(ctx context.Context, userID int64, in CategoryInput) (*Category, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO categories (user_id, kind, name, icon, color, sort_order)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6,
			(SELECT COALESCE(max(sort_order), 0) + 1 FROM categories WHERE user_id=$1)))
		RETURNING id`,
		userID, in.Kind, in.Name, in.Icon, in.Color, in.SortOrder).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetCategory(ctx, userID, id)
}

func (s *Store) UpdateCategory(ctx context.Context, userID, id int64, in CategoryInput) (*Category, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE categories SET
			name = COALESCE($3, name),
			icon = COALESCE($4, icon),
			color = COALESCE($5, color),
			sort_order = COALESCE($6, sort_order),
			is_archived = COALESCE($7, is_archived)
		WHERE user_id=$1 AND id=$2`,
		userID, id, in.Name, in.Icon, in.Color, in.SortOrder, in.IsArchived)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}
	return s.GetCategory(ctx, userID, id)
}

// DeleteCategory faqat tranzaksiyasi yo'q kategoriyani o'chiradi; aks holda arxivlash kerak.
func (s *Store) DeleteCategory(ctx context.Context, userID, id int64) error {
	var used bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM transactions WHERE category_id=$1)`, id).Scan(&used); err != nil {
		return err
	}
	if used {
		return ErrCategoryInUse
	}
	tag, err := s.pool.Exec(ctx, `DELETE FROM categories WHERE user_id=$1 AND id=$2`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
