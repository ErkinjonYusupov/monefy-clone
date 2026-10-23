import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api } from './api'
import type { Period } from './format'
import type { Account, Category, Me } from './types'

interface AppState {
  me: Me
  accounts: Account[]
  categories: Category[]
  period: Period
  setPeriod: (p: Period) => void
  accountId: number // 0 = barcha hisoblar
  setAccountId: (id: number) => void
  // revision har bir o'zgarishdan keyin oshadi; sahifalar shunga qarab qayta yuklaydi
  revision: number
  refresh: () => Promise<void>
  setMe: (me: Me) => void
}

const Ctx = createContext<AppState | null>(null)

export function useApp(): AppState {
  const v = useContext(Ctx)
  if (!v) throw new Error('useApp outside AppProvider')
  return v
}

export function AppProvider({ children }: { children: ReactNode }) {
  const [me, setMe] = useState<Me | null>(null)
  const [accounts, setAccounts] = useState<Account[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [period, setPeriod] = useState<Period>({ type: 'month', offset: 0 })
  const [accountId, setAccountId] = useState(0)
  const [revision, setRevision] = useState(0)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    const [m, a, c] = await Promise.all([api.me(), api.accounts(), api.categories()])
    setMe(m)
    setAccounts(a)
    setCategories(c)
    setRevision((r) => r + 1)
  }, [])

  useEffect(() => {
    refresh().catch((e) => setError(e instanceof Error ? e.message : String(e)))
  }, [refresh])

  const value = useMemo(
    () => me && { me, accounts, categories, period, setPeriod, accountId, setAccountId, revision, refresh, setMe },
    [me, accounts, categories, period, accountId, revision, refresh],
  )

  if (error) {
    return (
      <div className="center-screen">
        <p>Ulanib bo'lmadi</p>
        <p className="muted">{error}</p>
        <button className="btn" onClick={() => location.reload()}>Qayta urinish</button>
      </div>
    )
  }
  if (!value) return <div className="center-screen muted">Yuklanmoqda…</div>
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}
