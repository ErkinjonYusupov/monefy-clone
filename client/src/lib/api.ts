import { initData } from './telegram'
import type {
  Account, AdminStats, AdminUserDetail, AdminUserPage, Category, Me, PremiumInfo, Rate, Summary,
  Transaction, Transfer,
} from './types'

const BASE = import.meta.env.VITE_API_URL ?? '/api'

export class ApiError extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {}
  const data = initData()
  if (data) headers.Authorization = `tma ${data}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (res.status === 204) return undefined as T
  const json = await res.json().catch(() => null)
  if (!res.ok) {
    const err = json?.error
    throw new ApiError(res.status, err?.code ?? 'unknown', err?.message ?? `HTTP ${res.status}`)
  }
  return json as T
}

const qs = (params: Record<string, string | number | undefined>) => {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) if (v !== undefined && v !== '' && v !== 0) p.set(k, String(v))
  const s = p.toString()
  return s ? `?${s}` : ''
}

export type PeriodQuery = { from?: string; to?: string }

export const api = {
  me: () => request<Me>('GET', '/me'),
  savePhone: (response: string) => request<Me>('POST', '/me/phone', { response }),
  updateSettings: (body: { base_currency?: string; budget_amount?: number; clear_budget?: boolean }) =>
    request<Me>('PATCH', '/me/settings', body),

  accounts: () => request<Account[]>('GET', '/accounts'),
  createAccount: (body: Partial<Account>) => request<Account>('POST', '/accounts', body),
  updateAccount: (id: number, body: Partial<Account>) => request<Account>('PATCH', `/accounts/${id}`, body),
  deleteAccount: (id: number) => request<void>('DELETE', `/accounts/${id}`),

  categories: () => request<Category[]>('GET', '/categories'),
  createCategory: (body: Partial<Category>) => request<Category>('POST', '/categories', body),
  updateCategory: (id: number, body: Partial<Category>) => request<Category>('PATCH', `/categories/${id}`, body),
  deleteCategory: (id: number) => request<void>('DELETE', `/categories/${id}`),

  transactions: (q: PeriodQuery & { account_id?: number; category_id?: number; kind?: string; limit?: number }) =>
    request<Transaction[]>('GET', '/transactions' + qs(q)),
  transaction: (id: number) => request<Transaction>('GET', `/transactions/${id}`),
  createTransaction: (body: TxBody) => request<Transaction>('POST', '/transactions', body),
  updateTransaction: (id: number, body: Partial<TxBody>) => request<Transaction>('PATCH', `/transactions/${id}`, body),
  deleteTransaction: (id: number) => request<void>('DELETE', `/transactions/${id}`),

  transfers: (q: PeriodQuery) => request<Transfer[]>('GET', '/transfers' + qs(q)),
  createTransfer: (body: Omit<Transfer, 'id' | 'created_at'>) => request<Transfer>('POST', '/transfers', body),
  deleteTransfer: (id: number) => request<void>('DELETE', `/transfers/${id}`),

  summary: (q: PeriodQuery & { account_id?: number }) => request<Summary>('GET', '/summary' + qs(q)),

  rates: () => request<Rate[]>('GET', '/rates'),
  setRate: (currency: string, rate: string) => request<Rate[]>('PUT', `/rates/${currency}`, { rate }),
  deleteRate: (currency: string) => request<void>('DELETE', `/rates/${currency}`),

  exportCsv: (q: PeriodQuery) => request<{ rows: number }>('POST', '/export' + qs(q)),

  premium: () => request<PremiumInfo>('GET', '/premium'),
  requestPremium: (plan: string) => request<{ ok: boolean }>('POST', '/premium/request', { plan }),

  admin: {
    stats: () => request<AdminStats>('GET', '/admin/stats'),
    users: (q: string, page: number) => request<AdminUserPage>('GET', '/admin/users' + qs({ q, page })),
    user: (id: number) => request<AdminUserDetail>('GET', `/admin/users/${id}`),
    grant: (id: number, plan: string, amount?: number) =>
      request<AdminUserDetail>('POST', `/admin/users/${id}/premium`, { plan, amount }),
    revoke: (id: number) => request<AdminUserDetail>('POST', `/admin/users/${id}/revoke`),
  },
}

export type TxBody = {
  account_id: number
  category_id: number
  amount: number
  note?: string
  occurred_on?: string
}
