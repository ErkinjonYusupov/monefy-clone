import { ChevronLeft, ChevronRight, Crown, Search, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { MoneyInput, Page } from '../components/ui'
import { api } from '../lib/api'
import { formatMoney, parseMoney } from '../lib/format'
import { alertDialog, confirmDialog, haptic } from '../lib/telegram'
import type { AdminStats, AdminUser, AdminUserDetail, AdminUserPage } from '../lib/types'
import { untilLabel } from './Menu'

const fullName = (u: AdminUser) => `${u.first_name} ${u.last_name}`.trim() || `ID ${u.telegram_id}`
const dateLabel = (s: string) => new Date(s).toLocaleDateString('uz-UZ')

export function AdminHome() {
  const nav = useNavigate()
  const [stats, setStats] = useState<AdminStats | null>(null)

  useEffect(() => {
    api.admin.stats().then(setStats).catch((e) => alertDialog(e.message))
  }, [])

  return (
    <Page title="Admin panel">
      {stats && (
        <div className="stat-grid">
          <div className="stat"><small>Foydalanuvchilar</small><strong>{stats.users.total}</strong></div>
          <div className="stat"><small>Premium</small><strong>{stats.users.premium}</strong></div>
          <div className="stat"><small>Bugun yangi</small><strong>{stats.users.new_today}</strong></div>
          <div className="stat"><small>7 kunda faol</small><strong>{stats.users.active_week}</strong></div>
          <div className="stat wide">
            <small>Shu oy tushum ({stats.revenue.month_count} ta obuna)</small>
            <strong>{formatMoney(stats.revenue.month_amount, 'UZS')}</strong>
          </div>
        </div>
      )}
      <button className="admin-card" onClick={() => nav('/admin/users')}>
        <Users size={24} />
        <span className="grow"><strong>Foydalanuvchilar</strong><small>Qidirish, premium berish</small></span>
        <ChevronRight size={18} />
      </button>
    </Page>
  )
}

export function AdminUsers() {
  const nav = useNavigate()
  // Qidiruv va sahifa URL'da: foydalanuvchi kartasidan orqaga qaytganda saqlanib qoladi
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const page = Math.max(1, Number(params.get('page')) || 1)
  const [input, setInput] = useState(q)
  const [data, setData] = useState<AdminUserPage | null>(null)
  const [loading, setLoading] = useState(false)

  // Yozish to'xtagandan 350 ms keyin qidiriladi
  useEffect(() => {
    if (input === q) return
    const t = setTimeout(() => setParams(input ? { q: input } : {}, { replace: true }), 350)
    return () => clearTimeout(t)
  }, [input, q, setParams])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    api.admin.users(q, page)
      .then((d) => { if (!cancelled) setData(d) })
      .catch((e) => alertDialog(e.message))
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [q, page])

  const go = (p: number) => setParams({ ...(q ? { q } : {}), page: String(p) }, { replace: true })

  return (
    <Page title="Foydalanuvchilar">
      <label className="search">
        <Search size={18} className="muted" />
        <input value={input} onChange={(e) => setInput(e.target.value)} placeholder="Ism, username yoki telefon" />
      </label>
      {data && <p className="muted small">Topildi: {data.total}{loading && ' · yuklanmoqda…'}</p>}

      <ul className="list">
        {data?.items.map((u) => (
          <li key={u.id}>
            <button className="list-row" onClick={() => nav(`/admin/users/${u.id}`)}>
              <span className="grow">
                <span className="user-name">
                  {fullName(u)}
                  {u.is_premium && <Crown size={14} className="gold" />}
                </span>
                <small className="muted">
                  {u.username ? `@${u.username}` : `ID ${u.telegram_id}`}
                  {' · '}
                  {u.phone || "raqam yo'q"}
                </small>
              </span>
              <ChevronRight size={18} className="muted" />
            </button>
          </li>
        ))}
      </ul>
      {data && data.items.length === 0 && <p className="empty">Hech kim topilmadi</p>}

      {data && data.pages > 1 && (
        <div className="pager">
          <button className="btn" disabled={page <= 1} onClick={() => go(page - 1)}><ChevronLeft size={18} /></button>
          <span>{page} / {data.pages}</span>
          <button className="btn" disabled={page >= data.pages} onClick={() => go(page + 1)}><ChevronRight size={18} /></button>
        </div>
      )}
    </Page>
  )
}

export function AdminUserView() {
  const { id } = useParams()
  const [data, setData] = useState<AdminUserDetail | null>(null)
  const [plan, setPlan] = useState('month')
  const [amount, setAmount] = useState('')
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    api.admin.user(Number(id)).then(setData).catch((e) => alertDialog(e.message))
  }, [id])

  // Muddat tanlanganda summa reja narxi bilan to'ldiriladi (admin o'zgartirishi mumkin)
  useEffect(() => {
    const opt = data?.options.find((o) => o.code === plan)
    setAmount(opt && opt.price ? String(opt.price / 100) : '')
  }, [plan, data?.options])

  if (!data) return <div className="center-screen muted">Yuklanmoqda…</div>
  const u = data.user
  const opt = data.options.find((o) => o.code === plan)

  const grant = async () => {
    const minor = amount ? parseMoney(amount) : 0
    if (minor === null) return alertDialog("Summa noto'g'ri")
    if (!(await confirmDialog(`${fullName(u)} ga ${opt?.title} premium berilsinmi? Summa: ${formatMoney(minor, 'UZS')}`))) return
    setBusy(true)
    try {
      setData(await api.admin.grant(u.id, plan, Math.round(minor / 100)))
      haptic('success')
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }

  const revoke = async () => {
    if (!(await confirmDialog('Premium bekor qilinsinmi?'))) return
    try {
      setData(await api.admin.revoke(u.id))
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <Page title={fullName(u)}>
      <ul className="list info">
        <li className="list-row"><span className="grow muted">Username</span><span>{u.username ? `@${u.username}` : '—'}</span></li>
        <li className="list-row"><span className="grow muted">Telefon</span>
          {u.phone ? <a href={`tel:${u.phone}`}>{u.phone}</a> : <span>—</span>}
        </li>
        <li className="list-row"><span className="grow muted">Telegram ID</span><span>{u.telegram_id}</span></li>
        <li className="list-row"><span className="grow muted">Ro'yxatdan o'tgan</span><span>{dateLabel(u.created_at)}</span></li>
        <li className="list-row"><span className="grow muted">Oxirgi faollik</span><span>{dateLabel(u.last_seen_at)}</span></li>
        <li className="list-row"><span className="grow muted">Premium</span>
          <span className={u.is_premium ? 'gold' : 'muted'}>{u.is_premium ? untilLabel(u.premium_until) : "yo'q"}</span>
        </li>
      </ul>

      <h2 className="section-title">{u.is_premium ? 'Premiumni uzaytirish' : 'Premium berish'}</h2>
      <div className="grant-options">
        {data.options.map((o) => (
          <button key={o.code} className={o.code === plan ? 'selected' : ''} onClick={() => setPlan(o.code)}>{o.title}</button>
        ))}
      </div>
      <label className="field">
        <span>To'langan summa (so'm, bepul bo'lsa bo'sh)</span>
        <MoneyInput value={amount} onChange={setAmount} decimals={0} placeholder="0" />
      </label>
      <button className="btn primary wide" disabled={busy} onClick={grant}>
        <Crown size={18} /> {opt?.title} premium berish
      </button>
      {u.is_premium && <button className="btn wide danger" onClick={revoke}>Premiumni bekor qilish</button>}

      {data.subscriptions.length > 0 && (
        <>
          <h2 className="section-title">Obunalar tarixi</h2>
          <ul className="list">
            {data.subscriptions.map((s) => (
              <li key={s.id} className="list-row">
                <span className="grow">
                  {data.options.find((o) => o.code === s.plan)?.title ?? s.plan}
                  <small className="muted"> · {dateLabel(s.starts_at)} – {untilLabel(s.expires_at).replace(' gacha', '')}</small>
                </span>
                <span className={s.revoked_at ? 'muted strike' : ''}>{formatMoney(s.amount, 'UZS')}</span>
              </li>
            ))}
          </ul>
        </>
      )}
    </Page>
  )
}
