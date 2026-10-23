import { Menu, Minus, Plus } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CategoryBadge, Donut, PeriodBar } from '../components/ui'
import { api } from '../lib/api'
import { useCached } from '../lib/cache'
import { formatMoney, periodRange } from '../lib/format'
import { useApp } from '../lib/store'

export default function Home() {
  const { me, accounts, categories, period, accountId, setAccountId, revision } = useApp()
  const nav = useNavigate()
  const range = periodRange(period)
  const summary = useCached(`summary|${range.from}|${range.to}|${accountId}`, revision,
    () => api.summary({ from: range.from, to: range.to, account_id: accountId }))

  const totals = new Map(summary?.categories.map((c) => [c.category_id, c.total]))
  const expenseCats = categories.filter((c) => c.kind === 'expense' && !c.is_archived)
  const slices = expenseCats
    .map((c) => ({ value: totals.get(c.id) ?? 0, color: c.color }))
    .filter((s) => s.value > 0)

  const cur = summary?.currency ?? me.base_currency
  // Budjet asosiy valyutada, shuning uchun faqat "Barcha hisoblar"da ko'rsatiladi
  const budgetLeft = me.budget_amount != null && period.type === 'month' && accountId === 0 && summary
    ? me.budget_amount - summary.expense
    : null
  // Bir nechta valyuta bo'lsa, har biri alohida ko'rsatiladi (kursga o'girilmasdan)
  const byCur = summary?.by_currency ?? []
  const multi = byCur.length > 1
  const active = byCur.filter((c) => c.income || c.expense)

  // Kategoriya ikonkalari donut atrofida aylana bo'ylab joylashadi (Monefy kabi).
  // Aylana o'rtadagi bo'sh joyga (kenglik va balandlik) sig'adigan qilib o'lchanadi,
  // shunda tugmalar va balans qatori har doim ekranda qoladi.
  const areaRef = useRef<HTMLDivElement>(null)
  const area = useElementSize(areaRef)
  const ringSize = Math.max(200, Math.min(area.width - 8, area.height - 4, 400))
  const donutSize = ringSize * 0.56
  // Kategoriya ko'p bo'lsa ikonkalar ustma-ust tushmasligi uchun kichrayadi
  const approxRadius = ringSize / 2 - 22
  const badge = Math.max(24, Math.min(40, Math.floor((2 * Math.PI * approxRadius) / Math.max(expenseCats.length, 1)) - 6))
  const radius = ringSize / 2 - badge / 2 - 2
  // Markazdagi summalar halqa ichiga sig'ishi uchun shrift aylana bilan birga kichrayadi
  const centerFont = Math.max(11, Math.min(17, Math.round(donutSize * 0.085)))

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

      <div className="ring-area" ref={areaRef}>
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
                <CategoryBadge icon={c.icon} color={c.color} size={badge} />
              </button>
            )
          })}
          <div className="ring-center" onClick={() => nav('/transactions')}>
            <Donut slices={slices} size={donutSize} thickness={donutSize * 0.12}>
              <div style={{ fontSize: centerFont }}>
                {multi ? (
                  active.length === 0 ? <div className="muted">0</div> : active.map((c) => (
                    <div key={c.currency} className="cur-lines">
                      {c.income > 0 && <div className="income">+{formatMoney(c.income, c.currency)}</div>}
                      {c.expense > 0 && <div className="expense">−{formatMoney(c.expense, c.currency)}</div>}
                    </div>
                  ))
                ) : (
                  <>
                    <div className="income">{formatMoney(summary?.income ?? 0)}</div>
                    <div className="expense">{formatMoney(summary?.expense ?? 0)}</div>
                  </>
                )}
              </div>
            </Donut>
          </div>
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
        {multi ? (
          <>
            <strong className="cur-balances">
              {byCur.map((c) => (
                <span key={c.currency} className={c.balance < 0 ? 'neg' : ''}>{formatMoney(c.balance, c.currency)}</span>
              ))}
            </strong>
            <small className="muted">Jami ≈ {formatMoney(summary?.balance ?? 0, cur)}</small>
          </>
        ) : (
          <strong className={summary && summary.balance < 0 ? 'neg' : ''}>{formatMoney(summary?.balance ?? 0, cur)}</strong>
        )}
        {budgetLeft != null && (
          <small className={budgetLeft < 0 ? 'neg' : ''}>Budjet qoldig'i: {formatMoney(budgetLeft, cur)}</small>
        )}
      </button>
    </div>
  )
}

function useElementSize(ref: React.RefObject<HTMLElement | null>) {
  const [size, setSize] = useState({ width: window.innerWidth, height: window.innerWidth })
  useEffect(() => {
    const el = ref.current
    if (!el) return
    const ro = new ResizeObserver(([e]) => setSize({ width: e.contentRect.width, height: e.contentRect.height }))
    ro.observe(el)
    return () => ro.disconnect()
  }, [ref])
  return size
}
