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
