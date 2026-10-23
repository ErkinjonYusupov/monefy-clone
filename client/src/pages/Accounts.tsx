import { ArrowRightLeft, Plus, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Page, PremiumLock } from '../components/ui'
import { api } from '../lib/api'
import { CURRENCIES, currencyLabel, formatMoney, parseMoney, toInput, ymd } from '../lib/format'
import { ACCOUNT_ICONS, Icon } from '../lib/icons'
import { useApp } from '../lib/store'
import { alertDialog, confirmDialog, haptic } from '../lib/telegram'

export function Accounts() {
  const { accounts } = useApp()
  const nav = useNavigate()
  return (
    <Page
      title="Hisoblar"
      action={<button className="icon-btn" onClick={() => nav('/accounts/new')} aria-label="Qo'shish"><Plus size={22} /></button>}
    >
      <ul className="list">
        {accounts.map((a) => (
          <li key={a.id}>
            <button className={`list-row ${a.is_archived ? 'dim' : ''}`} onClick={() => nav(`/accounts/${a.id}`)}>
              <span className="badge neutral"><Icon name={a.icon} size={20} /></span>
              <span className="grow">{a.name}{a.is_archived && <small className="muted"> · arxiv</small>}</span>
              <strong className={a.balance < 0 ? 'neg' : ''}>{formatMoney(a.balance, a.currency)}</strong>
            </button>
          </li>
        ))}
      </ul>
      {accounts.length > 1 && (
        <button className="btn wide" onClick={() => nav('/transfer')}>
          <ArrowRightLeft size={18} /> Hisoblar orasida o'tkazma
        </button>
      )}
    </Page>
  )
}

export function AccountForm() {
  const { id } = useParams()
  const { me, accounts, refresh } = useApp()
  const nav = useNavigate()
  const existing = accounts.find((a) => a.id === Number(id))

  const [name, setName] = useState(existing?.name ?? '')
  const [currency, setCurrency] = useState(existing?.currency ?? me.base_currency)
  const [initial, setInitial] = useState(existing ? toInput(existing.initial_balance) : '')
  const [icon, setIcon] = useState(existing?.icon ?? 'wallet')
  const [archived, setArchived] = useState(existing?.is_archived ?? false)

  const save = async () => {
    const neg = initial.trim().startsWith('-')
    const parsed = initial.trim() ? parseMoney(initial.replace('-', '')) : 0
    if (parsed === null) return alertDialog("Boshlang'ich qoldiq noto'g'ri")
    const body = { name: name.trim(), currency, initial_balance: neg ? -parsed : parsed, icon, is_archived: archived }
    try {
      if (existing) await api.updateAccount(existing.id, body)
      else await api.createAccount(body)
      haptic('success')
      await refresh()
      nav(-1)
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    }
  }

  const remove = async () => {
    if (!existing) return
    if (!(await confirmDialog(`"${existing.name}" va undagi barcha yozuvlar o'chiriladi. Davom etasizmi?`))) return
    try {
      await api.deleteAccount(existing.id)
      await refresh()
      nav(-1)
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Page
      title={existing ? 'Hisobni tahrirlash' : 'Yangi hisob'}
      action={existing && <button className="icon-btn" onClick={remove} aria-label="O'chirish"><Trash2 size={20} /></button>}
    >
      <label className="field">
        <span>Nomi</span>
        <input value={name} maxLength={64} onChange={(e) => setName(e.target.value)} placeholder="Masalan: Humo karta" />
      </label>
      <label className="field">
        <span>Valyuta</span>
        <select value={currency} onChange={(e) => setCurrency(e.target.value)} disabled={!me.is_premium}>
          {CURRENCIES.map((c) => <option key={c} value={c}>{c}</option>)}
        </select>
      </label>
      {!me.is_premium && <PremiumLock text="Boshqa valyutadagi hisoblar — Premium" />}
      <label className="field">
        <span>Boshlang'ich qoldiq ({currencyLabel(currency)})</span>
        <input inputMode="decimal" value={initial} onChange={(e) => setInitial(e.target.value)} placeholder="0" />
      </label>
      <div className="field">
        <span>Ikonka</span>
        <div className="icon-pick">
          {ACCOUNT_ICONS.map((i) => (
            <button key={i} className={i === icon ? 'selected' : ''} onClick={() => setIcon(i)}><Icon name={i} /></button>
          ))}
        </div>
      </div>
      {existing && (
        <label className="check">
          <input type="checkbox" checked={archived} onChange={(e) => setArchived(e.target.checked)} />
          Arxivlangan (ro'yxatda ko'rinmaydi)
        </label>
      )}
      <button className="btn primary wide" onClick={save}>Saqlash</button>
    </Page>
  )
}

export function TransferForm() {
  const { accounts, refresh } = useApp()
  const nav = useNavigate()
  const active = accounts.filter((a) => !a.is_archived)
  const [from, setFrom] = useState(active[0]?.id ?? 0)
  const [to, setTo] = useState(active[1]?.id ?? 0)
  const [amountFrom, setAmountFrom] = useState('')
  const [amountTo, setAmountTo] = useState('')
  const [date, setDate] = useState(ymd(new Date()))
  const [note, setNote] = useState('')

  const fromAcc = accounts.find((a) => a.id === from)
  const toAcc = accounts.find((a) => a.id === to)
  const sameCurrency = fromAcc?.currency === toAcc?.currency

  useEffect(() => {
    if (sameCurrency) setAmountTo('')
  }, [sameCurrency])

  const save = async () => {
    if (from === to) return alertDialog('Hisoblar har xil bo\'lishi kerak')
    const a = parseMoney(amountFrom)
    const b = sameCurrency ? a : parseMoney(amountTo)
    if (!a || !b) return alertDialog('Summani kiriting')
    try {
      await api.createTransfer({ from_account_id: from, to_account_id: to, amount_from: a, amount_to: b, note, occurred_on: date })
      haptic('success')
      await refresh()
      nav(-1)
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Page title="O'tkazma">
      <label className="field">
        <span>Qayerdan</span>
        <select value={from} onChange={(e) => setFrom(Number(e.target.value))}>
          {active.map((a) => <option key={a.id} value={a.id}>{a.name} ({formatMoney(a.balance, a.currency)})</option>)}
        </select>
      </label>
      <label className="field">
        <span>Qayerga</span>
        <select value={to} onChange={(e) => setTo(Number(e.target.value))}>
          {active.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
        </select>
      </label>
      <label className="field">
        <span>Summa ({currencyLabel(fromAcc?.currency ?? '')})</span>
        <input inputMode="decimal" value={amountFrom} onChange={(e) => setAmountFrom(e.target.value)} placeholder="0" />
      </label>
      {!sameCurrency && (
        <label className="field">
          <span>Tushadigan summa ({currencyLabel(toAcc?.currency ?? '')})</span>
          <input inputMode="decimal" value={amountTo} onChange={(e) => setAmountTo(e.target.value)} placeholder="0" />
        </label>
      )}
      <label className="field">
        <span>Sana</span>
        <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
      </label>
      <label className="field">
        <span>Izoh</span>
        <input value={note} maxLength={500} onChange={(e) => setNote(e.target.value)} />
      </label>
      <button className="btn primary wide" onClick={save}>O'tkazish</button>
    </Page>
  )
}
