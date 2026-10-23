import { ChevronRight, Coins, Crown, FileDown, List, Shapes, ShieldCheck, Wallet } from 'lucide-react'
import { useEffect, useState, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { MoneyInput, Page, PremiumLock } from '../components/ui'
import { api } from '../lib/api'
import { CURRENCIES, parseMoney, periodRange, toInput } from '../lib/format'
import { useApp } from '../lib/store'
import { alertDialog, confirmDialog, haptic } from '../lib/telegram'
import type { Rate } from '../lib/types'

function Row({ icon, label, value, onClick }: { icon: ReactNode; label: string; value?: ReactNode; onClick: () => void }) {
  return (
    <li>
      <button className="list-row" onClick={onClick}>
        {icon}
        <span className="grow">{label}</span>
        {value && <span className="muted">{value}</span>}
        <ChevronRight size={18} className="muted" />
      </button>
    </li>
  )
}

export function Menu() {
  const { me, setMe, period, refresh } = useApp()
  const nav = useNavigate()
  const [budget, setBudget] = useState(me.budget_amount ? toInput(me.budget_amount) : '')

  const changeCurrency = async (cur: string) => {
    if (cur === me.base_currency) return
    const msg = me.is_premium
      ? `Asosiy valyuta ${cur} bo'ladi. Boshqa valyutalar kurslarini qayta tekshiring.`
      : `${me.base_currency} dagi hisoblar valyutasi ${cur} ga o'zgaradi (summalar o'zgarmaydi). Davom etasizmi?`
    if (!(await confirmDialog(msg))) return
    setMe(await api.updateSettings({ base_currency: cur }))
    await refresh()
  }

  const saveBudget = async () => {
    if (!budget.trim()) {
      setMe(await api.updateSettings({ clear_budget: true }))
      return
    }
    const v = parseMoney(budget)
    if (!v) return alertDialog("Budjet summasi noto'g'ri")
    setMe(await api.updateSettings({ budget_amount: v }))
    haptic('success')
  }

  const exportCsv = async () => {
    const r = periodRange(period)
    try {
      const res = await api.exportCsv({ from: r.from, to: r.to })
      alertDialog(`CSV fayl botga yuborildi (${res.rows} ta yozuv)`)
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Page title="Menyu">
      <button className={`premium-card ${me.is_premium ? 'active' : ''}`} onClick={() => nav('/premium')}>
        <Crown size={26} />
        <div className="grow">
          <strong>{me.is_premium ? 'Premium faol' : 'Premium olish'}</strong>
          <small>{me.is_premium ? untilLabel(me.premium_until) : "Ko'p valyuta, kategoriyalar, eksport"}</small>
        </div>
        <ChevronRight size={18} />
      </button>

      {me.is_admin && (
        <button className="premium-card admin" onClick={() => nav('/admin')}>
          <ShieldCheck size={26} />
          <div className="grow">
            <strong>Admin panel</strong>
            <small>Foydalanuvchilar va obunalar</small>
          </div>
          <ChevronRight size={18} />
        </button>
      )}

      <ul className="list">
        <Row icon={<List size={20} />} label="Yozuvlar" onClick={() => nav('/transactions')} />
        <Row icon={<Wallet size={20} />} label="Hisoblar" onClick={() => nav('/accounts')} />
        <Row icon={<Shapes size={20} />} label="Kategoriyalar" onClick={() => nav('/categories')} />
        <Row icon={<Coins size={20} />} label="Valyuta kurslari" onClick={() => nav('/rates')} />
      </ul>

      <h2 className="section-title">Sozlamalar</h2>
      <label className="field">
        <span>Asosiy valyuta</span>
        <select value={me.base_currency} onChange={(e) => changeCurrency(e.target.value)}>
          {CURRENCIES.map((c) => <option key={c} value={c}>{c}</option>)}
        </select>
      </label>
      <label className="field">
        <span>Oylik budjet (bo'sh = o'chiq)</span>
        <div className="inline">
          <MoneyInput value={budget} onChange={setBudget} placeholder="Masalan 5 000 000" />
          <button className="btn" onClick={saveBudget}>Saqlash</button>
        </div>
      </label>

      <h2 className="section-title">Ma'lumotlar</h2>
      {me.is_premium ? (
        <button className="btn wide" onClick={exportCsv}>
          <FileDown size={18} /> Joriy davrni CSV ga eksport
        </button>
      ) : (
        <PremiumLock text="CSV eksport — Premium" />
      )}
    </Page>
  )
}

export function untilLabel(until: string | null): string {
  if (!until) return ''
  const d = new Date(until)
  if (d.getFullYear() - new Date().getFullYear() > 50) return 'Umrbod'
  return `${d.toLocaleDateString('uz-UZ')} gacha`
}

export function Rates() {
  const { me, accounts, revision, refresh } = useApp()
  const [rates, setRates] = useState<Rate[]>([])
  const [edit, setEdit] = useState<Record<string, string>>({})

  useEffect(() => {
    api.rates().then(setRates).catch(console.error)
  }, [revision])

  // Hisoblarda ishlatilgan va kursi kiritilgan barcha valyutalar.
  const currencies = [...new Set([...accounts.map((a) => a.currency), ...rates.map((r) => r.currency)])]
    .filter((c) => c !== me.base_currency)

  const save = async (cur: string) => {
    const v = (edit[cur] ?? '').replace(/\.$/, '')
    try {
      setRates(await api.setRate(cur, v))
      haptic('success')
      await refresh()
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Page title="Valyuta kurslari">
      {!me.is_premium && <PremiumLock text="Ko'p valyuta — Premium" />}
      <p className="muted">1 birlik valyuta necha {me.base_currency} ga teng.</p>
      {currencies.length === 0 && <p className="empty">Boshqa valyutadagi hisob yo'q</p>}
      {currencies.map((cur) => {
        const r = rates.find((x) => x.currency === cur)
        return (
          <label key={cur} className="field">
            <span>1 {cur} = ? {me.base_currency}</span>
            <div className="inline">
              <MoneyInput
                decimals={8}
                disabled={!me.is_premium}
                value={edit[cur] ?? (r ? String(Number(r.rate)) : '')}
                onChange={(v) => setEdit({ ...edit, [cur]: v })}
                placeholder="Kurs"
              />
              <button className="btn" disabled={!me.is_premium} onClick={() => save(cur)}>Saqlash</button>
            </div>
          </label>
        )
      })}
    </Page>
  )
}
