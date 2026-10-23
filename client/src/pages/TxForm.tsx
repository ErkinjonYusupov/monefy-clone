import { Delete, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { CategoryBadge } from '../components/ui'
import { api } from '../lib/api'
import { currencyLabel, parseMoney, toInput, ymd } from '../lib/format'
import { useApp } from '../lib/store'
import { alertDialog, confirmDialog, haptic } from '../lib/telegram'
import type { Kind } from '../lib/types'

const KEYS = ['7', '8', '9', '4', '5', '6', '1', '2', '3', '.', '0', 'del']

// TxForm yangi tranzaksiya (/add/:kind) va tahrirlash (/edit/:id) uchun.
export default function TxForm() {
  const { kind: kindParam, id } = useParams()
  const [search] = useSearchParams()
  const { accounts, categories, accountId, refresh } = useApp()
  const nav = useNavigate()

  const editId = id ? Number(id) : 0
  const [kind, setKind] = useState<Kind>(kindParam === 'income' ? 'income' : 'expense')
  const [amount, setAmount] = useState('')
  const activeAccounts = accounts.filter((a) => !a.is_archived)
  const [account, setAccount] = useState(accountId || activeAccounts[0]?.id || 0)
  const [category, setCategory] = useState(Number(search.get('category')) || 0)
  const [date, setDate] = useState(ymd(new Date()))
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!editId) return
    api.transaction(editId).then((t) => {
      setKind(t.kind)
      setAmount(toInput(t.amount))
      setAccount(t.account_id)
      setCategory(t.category_id)
      setDate(t.occurred_on)
      setNote(t.note)
    }).catch((e) => alertDialog(e.message))
  }, [editId])

  const cats = categories.filter((c) => c.kind === kind && (!c.is_archived || c.id === category))
  const currency = accounts.find((a) => a.id === account)?.currency ?? ''

  const press = (k: string) => {
    haptic()
    setAmount((a) => {
      if (k === 'del') return a.slice(0, -1)
      if (k === '.') return a.includes('.') ? a : (a || '0') + '.'
      const next = a === '0' ? k : a + k
      return parseMoney(next) === null || next.length > 13 ? a : next
    })
  }

  const save = async () => {
    const minor = parseMoney(amount)
    if (!minor) return alertDialog('Summani kiriting')
    if (!category) return alertDialog('Kategoriyani tanlang')
    if (!account) return alertDialog('Hisobni tanlang')
    setSaving(true)
    try {
      const body = { account_id: account, category_id: category, amount: minor, note: note.trim(), occurred_on: date }
      if (editId) await api.updateTransaction(editId, body)
      else await api.createTransaction(body)
      haptic('success')
      await refresh()
      nav(-1)
    } catch (e) {
      haptic('error')
      alertDialog(e instanceof Error ? e.message : String(e))
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!(await confirmDialog("Yozuvni o'chirasizmi?"))) return
    await api.deleteTransaction(editId)
    await refresh()
    nav(-1)
  }

  return (
    <div className={`tx-form kind-${kind}`}>
      <header className="tx-header">
        <div className="seg">
          <button className={kind === 'expense' ? 'active' : ''} onClick={() => { setKind('expense'); setCategory(0) }}>Xarajat</button>
          <button className={kind === 'income' ? 'active' : ''} onClick={() => { setKind('income'); setCategory(0) }}>Daromad</button>
        </div>
        {editId > 0 && (
          <button className="icon-btn" onClick={remove} aria-label="O'chirish"><Trash2 size={20} /></button>
        )}
      </header>

      <div className="amount-display">
        <span>{amount || '0'}</span>
        <small>{currencyLabel(currency)}</small>
      </div>

      <div className="row-fields">
        <select value={account} onChange={(e) => setAccount(Number(e.target.value))}>
          {activeAccounts.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
        </select>
        <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
      </div>
      <input className="note" placeholder="Izoh" value={note} maxLength={500} onChange={(e) => setNote(e.target.value)} />

      <div className="cat-grid">
        {cats.map((c) => (
          <button key={c.id} className={`cat-cell ${c.id === category ? 'selected' : ''}`} onClick={() => { haptic(); setCategory(c.id) }}>
            <CategoryBadge icon={c.icon} color={c.color} size={42} />
            <span>{c.name}</span>
          </button>
        ))}
      </div>

      <div className="keypad">
        {KEYS.map((k) => (
          <button key={k} onClick={() => press(k)}>{k === 'del' ? <Delete size={22} /> : k}</button>
        ))}
      </div>
      <button className="btn primary save" disabled={saving} onClick={save}>
        {editId ? 'Saqlash' : kind === 'expense' ? "Xarajat qo'shish" : "Daromad qo'shish"}
      </button>
    </div>
  )
}
