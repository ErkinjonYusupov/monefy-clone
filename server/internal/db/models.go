package db

import "time"

type User struct {
	ID           int64      `json:"id"`
	TelegramID   int64      `json:"telegram_id"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Username     string     `json:"username"`
	LanguageCode string     `json:"language_code"`
	BaseCurrency string     `json:"base_currency"`
	BudgetAmount *int64     `json:"budget_amount"`
	PremiumUntil *time.Time `json:"premium_until"`
	CreatedAt    time.Time  `json:"created_at"`
	LastSeenAt   time.Time  `json:"last_seen_at"`
	Phone        string     `json:"phone"`
}

func (u *User) IsPremium() bool {
	return u.PremiumUntil != nil && u.PremiumUntil.After(time.Now())
}

type Account struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Currency       string `json:"currency"`
	InitialBalance int64  `json:"initial_balance"`
	Icon           string `json:"icon"`
	IsArchived     bool   `json:"is_archived"`
	Balance        int64  `json:"balance"` // hisob valyutasida
}

type Category struct {
	ID         int64  `json:"id"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	Color      string `json:"color"`
	SortOrder  int    `json:"sort_order"`
	IsArchived bool   `json:"is_archived"`
}

type Transaction struct {
	ID         int64     `json:"id"`
	AccountID  int64     `json:"account_id"`
	CategoryID int64     `json:"category_id"`
	Kind       string    `json:"kind"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	Note       string    `json:"note"`
	OccurredOn string    `json:"occurred_on"` // YYYY-MM-DD
	CreatedAt  time.Time `json:"created_at"`
}

type Transfer struct {
	ID            int64     `json:"id"`
	FromAccountID int64     `json:"from_account_id"`
	ToAccountID   int64     `json:"to_account_id"`
	AmountFrom    int64     `json:"amount_from"`
	AmountTo      int64     `json:"amount_to"`
	Note          string    `json:"note"`
	OccurredOn    string    `json:"occurred_on"`
	CreatedAt     time.Time `json:"created_at"`
}

type Rate struct {
	Currency  string    `json:"currency"`
	Rate      string    `json:"rate"` // NUMERIC, aniqlikni yo'qotmaslik uchun string
	UpdatedAt time.Time `json:"updated_at"`
}

type Subscription struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Plan      string     `json:"plan"`
	Source    string     `json:"source"`
	Amount    int64      `json:"amount"`
	Currency  string     `json:"currency"`
	GrantedBy *int64     `json:"granted_by"`
	StartsAt  time.Time  `json:"starts_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	Note      string     `json:"note"`
	CreatedAt time.Time  `json:"created_at"`
}

// Summary tanlangan davr bo'yicha natija, summalar asosiy valyutada.
type Summary struct {
	Currency     string          `json:"currency"`
	Income       int64           `json:"income"`
	Expense      int64           `json:"expense"`
	Balance      int64           `json:"balance"` // davr oxirigacha jami qoldiq
	Categories   []CategoryTotal `json:"categories"`
	MissingRates []string        `json:"missing_rates"`
	ByCurrency   []CurrencyTotal `json:"by_currency"`
}

// CurrencyTotal bitta valyuta bo'yicha, o'girilmagan summalar.
type CurrencyTotal struct {
	Currency string `json:"currency"`
	Income   int64  `json:"income"`
	Expense  int64  `json:"expense"`
	Balance  int64  `json:"balance"`
}

type CategoryTotal struct {
	CategoryID int64  `json:"category_id"`
	Kind       string `json:"kind"`
	Total      int64  `json:"total"`
	Count      int    `json:"count"`
}
