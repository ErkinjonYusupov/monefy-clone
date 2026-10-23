export type Kind = 'expense' | 'income'

export interface Me {
  id: number
  telegram_id: number
  first_name: string
  username: string
  base_currency: string
  budget_amount: number | null
  premium_until: string | null
  is_premium: boolean
  is_admin: boolean
  phone: string
  phone_required: boolean
}

export interface Account {
  id: number
  name: string
  currency: string
  initial_balance: number
  icon: string
  is_archived: boolean
  balance: number
}

export interface Category {
  id: number
  kind: Kind
  name: string
  icon: string
  color: string
  sort_order: number
  is_archived: boolean
}

export interface Transaction {
  id: number
  account_id: number
  category_id: number
  kind: Kind
  amount: number
  currency: string
  note: string
  occurred_on: string
  created_at: string
}

export interface Transfer {
  id: number
  from_account_id: number
  to_account_id: number
  amount_from: number
  amount_to: number
  note: string
  occurred_on: string
  created_at: string
}

export interface Rate {
  currency: string
  rate: string
  updated_at: string
}

export interface CategoryTotal {
  category_id: number
  kind: Kind
  total: number
  count: number
}

export interface Summary {
  currency: string
  income: number
  expense: number
  balance: number
  categories: CategoryTotal[]
  missing_rates: string[]
  by_currency: CurrencyTotal[]
}

// Bitta valyuta bo'yicha, kursga o'girilmagan summalar
export interface CurrencyTotal {
  currency: string
  income: number
  expense: number
  balance: number
}

export interface Plan {
  code: string
  title: string
  days: number
  price: number
}

export interface Subscription {
  id: number
  plan: string
  source: string
  amount: number
  starts_at: string
  expires_at: string
  revoked_at: string | null
}

export interface PremiumInfo {
  is_premium: boolean
  premium_until: string | null
  plans: Plan[]
  features: string[]
  subscriptions: Subscription[]
}

export interface AdminUser {
  id: number
  telegram_id: number
  first_name: string
  last_name: string
  username: string
  phone: string
  premium_until: string | null
  is_premium: boolean
  created_at: string
  last_seen_at: string
}

export interface AdminUserPage {
  items: AdminUser[]
  total: number
  page: number
  pages: number
}

export interface AdminUserDetail {
  user: AdminUser
  subscriptions: Subscription[]
  options: Plan[]
}

export interface AdminStats {
  users: { total: number; premium: number; new_today: number; active_week: number }
  revenue: { month_count: number; month_amount: number; total_count: number; total_amount: number }
}
