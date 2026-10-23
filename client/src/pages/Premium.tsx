import { Check, Crown } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Page } from '../components/ui'
import { api } from '../lib/api'
import { formatMoney } from '../lib/format'
import { useApp } from '../lib/store'
import { alertDialog, haptic } from '../lib/telegram'
import type { PremiumInfo } from '../lib/types'
import { untilLabel } from './Menu'

export default function Premium() {
  const { revision } = useApp()
  const [info, setInfo] = useState<PremiumInfo | null>(null)
  const [plan, setPlan] = useState('year')
  const [sent, setSent] = useState(false)

  useEffect(() => {
    api.premium().then(setInfo).catch(console.error)
  }, [revision])

  // Click ulanguncha so'rov adminga boradi, admin /grant bilan faollashtiradi.
  const request = async () => {
    try {
      await api.requestPremium(plan)
      haptic('success')
      setSent(true)
      alertDialog("So'rovingiz yuborildi. To'lov bo'yicha admin siz bilan bog'lanadi.")
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    }
  }

  if (!info) return <div className="center-screen muted">Yuklanmoqda…</div>

  return (
    <Page title="Premium">
      <div className="premium-hero">
        <Crown size={44} />
        {info.is_premium ? <p>Premium faol: <strong>{untilLabel(info.premium_until)}</strong></p> : <p>Barcha imkoniyatlarni oching</p>}
      </div>

      <ul className="features">
        {info.features.map((f) => <li key={f}><Check size={18} /> {f}</li>)}
      </ul>

      <div className="plans">
        {info.plans.map((p) => (
          <button key={p.code} className={`plan ${plan === p.code ? 'selected' : ''}`} onClick={() => setPlan(p.code)}>
            <span>{p.title}</span>
            <strong>{formatMoney(p.price, 'UZS')}</strong>
          </button>
        ))}
      </div>

      <button className="btn primary wide" disabled={sent} onClick={request}>
        {sent ? "So'rov yuborildi" : info.is_premium ? 'Muddatni uzaytirish' : 'Premium olish'}
      </button>

      {info.subscriptions.length > 0 && (
        <>
          <h2 className="section-title">Obunalar tarixi</h2>
          <ul className="list">
            {info.subscriptions.map((s) => (
              <li key={s.id} className="list-row">
                <span className="grow">
                  {info.plans.find((p) => p.code === s.plan)?.title ?? s.plan}
                  <small className="muted"> · {new Date(s.starts_at).toLocaleDateString('uz-UZ')} – {untilLabel(s.expires_at).replace(' gacha', '')}</small>
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
