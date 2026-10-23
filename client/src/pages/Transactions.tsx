import { ArrowRightLeft, ChevronDown, Trash2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { CategoryBadge, Page, PeriodBar } from '../components/ui'
import { api } from '../lib/api'
import { formatMoney, humanDate, periodRange } from '../lib/format'
import { useApp } from '../lib/store'
import { confirmDialog } from '../lib/telegram'
import type { Transaction, Transfer } from '../lib/types'

// Monefy kabi: yozuvlar kategoriya bo'yicha guruhlanadi, bosilganda ochiladi.
export default function Transactions() {
  const { categories, accounts, period, accountId, revision, refresh } = useApp()
  const nav = useNavigate()
  const range = periodRange(period)
  const [txs, setTxs] = useState<Transaction[]>([])
  const [transfers, setTransfers] = useState<Transfer[]>([])
  const [open, setOpen] = useState<number | null>(null)

  useEffect(() => {
    const q = { from: range.from, to: range.to }
    api.transactions({ ...q, account_id: accountId, limit: 1000 }).then(setTxs).catch(console.error)
    api.transfers(q).then(setTransfers).catch(console.error)
  }, [range.from, range.to, accountId, revision])

  const catById = useMemo(() => new Map(categories.map((c) => [c.id, c])), [categories])
  const accById = useMemo(() => new Map(accounts.map((a) => [a.id, a])), [accounts])

  const groups = useMemo(() => {
    const m = new Map<number, Transaction[]>()
    for (const t of txs) m.set(t.category_id, [...(m.get(t.category_id) ?? []), t])
    return [...m.entries()]
      .map(([id, items]) => ({ cat: catById.get(id)!, items, currency: items[0].currency, total: items.reduce((s, t) => s + t.amount, 0) }))
      .filter((g) => g.cat)
      .sort((a, b) => (a.cat.kind === b.cat.kind ? b.total - a.total : a.cat.kind === 'income' ? -1 : 1))
  }, [txs, catById])

  const visibleTransfers = transfers.filter((t) => !accountId || t.from_account_id === accountId || t.to_account_id === accountId)

  const removeTransfer = async (id: number) => {
    if (!(await confirmDialog("O'tkazmani o'chirasizmi?"))) return
    await api.deleteTransfer(id)
    await refresh()
  }

  // Bir guruhda bir nechta valyuta bo'lsa, jami ko'rsatilmaydi.
  const mixed = (items: Transaction[]) => new Set(items.map((t) => t.currency)).size > 1

  return (
    <Page title="Yozuvlar">
      <PeriodBar />
      {groups.length === 0 && visibleTransfers.length === 0 && <p className="empty">Bu davrda yozuvlar yo'q</p>}

      <ul className="list">
        {groups.map((g) => (
          <li key={g.cat.id}>
            <button className="list-row" onClick={() => setOpen(open === g.cat.id ? null : g.cat.id)}>
              <CategoryBadge icon={g.cat.icon} color={g.cat.color} size={38} />
              <span className="grow">{g.cat.name} <small className="muted">· {g.items.length}</small></span>
              <strong className={g.cat.kind}>{mixed(g.items) ? '—' : formatMoney(g.total, g.currency)}</strong>
              <ChevronDown size={18} className={open === g.cat.id ? 'rot' : ''} />
            </button>
            {open === g.cat.id && (
              <ul className="sublist">
                {g.items.map((t) => (
                  <li key={t.id}>
                    <button className="list-row sub" onClick={() => nav(`/edit/${t.id}`)}>
                      <span className="grow">
                        {humanDate(t.occurred_on)}
                        <small className="muted"> · {accById.get(t.account_id)?.name}</small>
                        {t.note && <div className="muted note-line">{t.note}</div>}
                      </span>
                      <span className={t.kind}>{formatMoney(t.amount, t.currency)}</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </li>
        ))}
      </ul>

      {visibleTransfers.length > 0 && (
        <>
          <h2 className="section-title">O'tkazmalar</h2>
          <ul className="list">
            {visibleTransfers.map((t) => {
              const from = accById.get(t.from_account_id)
              const to = accById.get(t.to_account_id)
              return (
                <li key={t.id} className="list-row">
                  <ArrowRightLeft size={20} className="muted" />
                  <span className="grow">
                    {from?.name} → {to?.name}
                    <small className="muted"> · {humanDate(t.occurred_on)}</small>
                  </span>
                  <span>{formatMoney(t.amount_from, from?.currency)}</span>
                  <button className="icon-btn" onClick={() => removeTransfer(t.id)} aria-label="O'chirish"><Trash2 size={16} /></button>
                </li>
              )
            })}
          </ul>
        </>
      )}
    </Page>
  )
}
