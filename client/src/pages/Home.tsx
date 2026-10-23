import { Menu, Minus, Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CategoryBadge, Donut, PeriodBar } from '../components/ui'
import { api } from '../lib/api'
import { formatMoney, periodRange } from '../lib/format'
import { useApp } from '../lib/store'
import type { Summary } from '../lib/types'

export default function Home() {
  const { me, accounts, categories, period, accountId, setAccountId, revision } = useApp()
  const nav = useNavigate()
  const [summary, setSummary] = useState<Summary | null>(null)
  const range = periodRange(period)

  useEffect(() => {
    api.summary({ from: range.from, to: range.to, account_id: accountId }).then(setSummary).catch(console.error)
  }, [range.from, range.to, accountId, revision])

  const totals = new Map(summary?.categories.map((c) => [c.category_id, c.total]))
  const expenseCats = categories.filter((c) => c.kind === 'expense' && !c.is_archived)
  const slices = expenseCats
    .map((c) => ({ value: totals.get(c.id) ?? 0, color: c.color }))
    .filter((s) => s.value > 0)

  const cur = summary?.currency ?? me.base_currency
  const budgetLeft = me.budget_amount != null && period.type === 'month' && summary
    ? me.budget_amount - summary.expense
    : null

  // Kategoriya ikonkalari donut atrofida aylana bo'ylab joylashadi (Monefy kabi).
  const ringSize = Math.min(window.innerWidth - 32, 380)
  const donutSize = ringSize * 0.56
  const radius = ringSize / 2 - 26

  return (
    <div className="home">
      <header className="home-header">
        <select className="account-select" value={accountId} onChange={(e) => setAccountId(Number(e.target.value))}>
          <option value={0}>Barcha hisoblar</option>
          {accounts.filter((a) => !a.is_archived).map((a) => (
            <option key={a.id} value={a.id}>{a.name}</option>
          ))}
        </select>
        <button className="icon-btn" onClick={() => nav('/menu')} aria-label="Menyu">
          <Menu size={22} />
        </button>
      </header>

      <PeriodBar />

      <div className="ring" style={{ width: ringSize, height: ringSize }}>
        {expenseCats.map((c, i) => {
          const angle = (i / expenseCats.length) * 2 * Math.PI - Math.PI / 2
          const x = ringSize / 2 + radius * Math.cos(angle)
          const y = ringSize / 2 + radius * Math.sin(angle)
          const has = (totals.get(c.id) ?? 0) > 0
          return (
            <button
              key={c.id}
              className={`ring-item ${has ? '' : 'dim'}`}
              style={{ left: x, top: y }}
              onClick={() => nav(`/add/expense?category=${c.id}`)}
              title={c.name}
            >
              <CategoryBadge icon={c.icon} color={c.color} size={40} />
            </button>
          )
        })}
        <div className="ring-center" onClick={() => nav('/transactions')}>
          <Donut slices={slices} size={donutSize} thickness={donutSize * 0.12}>
            <div className="income">{formatMoney(summary?.income ?? 0)}</div>
            <div className="expense">{formatMoney(summary?.expense ?? 0)}</div>
          </Donut>
        </div>
      </div>

      {summary && summary.missing_rates.length > 0 && (
        <p className="warning" onClick={() => nav('/rates')}>
          Kurs kiritilmagan: {summary.missing_rates.join(', ')}. Kiritish uchun bosing.
        </p>
      )}

      <div className="actions">
        <button className="action expense-btn" onClick={() => nav('/add/expense')} aria-label="Xarajat">
          <Minus size={34} />
        </button>
        <button className="action income-btn" onClick={() => nav('/add/income')} aria-label="Daromad">
          <Plus size={34} />
        </button>
      </div>

      <button className="balance-bar" onClick={() => nav('/transactions')}>
        <span>Balans</span>
        <strong className={summary && summary.balance < 0 ? 'neg' : ''}>{formatMoney(summary?.balance ?? 0, cur)}</strong>
        {budgetLeft != null && (
          <small className={budgetLeft < 0 ? 'neg' : ''}>Budjet qoldig'i: {formatMoney(budgetLeft, cur)}</small>
        )}
      </button>
    </div>
  )
}
