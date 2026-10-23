import { ChevronLeft, ChevronRight, Crown } from 'lucide-react'
import { useLayoutEffect, useRef, type ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import { cleanAmount, groupDigits, PERIOD_LABELS, periodRange, type PeriodType } from '../lib/format'
import { Icon } from '../lib/icons'
import { useApp } from '../lib/store'

export function CategoryBadge({ icon, color, size = 44 }: { icon: string; color: string; size?: number }) {
  return (
    <span className="badge" style={{ width: size, height: size, background: color }}>
      <Icon name={icon} size={Math.round(size * 0.5)} color="#fff" />
    </span>
  )
}

export function PeriodBar() {
  const { period, setPeriod } = useApp()
  const range = periodRange(period)
  return (
    <div className="period">
      <div className="period-tabs">
        {(Object.keys(PERIOD_LABELS) as PeriodType[]).map((t) => (
          <button key={t} className={t === period.type ? 'active' : ''} onClick={() => setPeriod({ type: t, offset: 0 })}>
            {PERIOD_LABELS[t]}
          </button>
        ))}
      </div>
      <div className="period-nav">
        <button className="icon-btn" disabled={period.type === 'all'} onClick={() => setPeriod({ ...period, offset: period.offset - 1 })}>
          <ChevronLeft size={20} />
        </button>
        <span>{range.label}</span>
        <button className="icon-btn" disabled={period.type === 'all'} onClick={() => setPeriod({ ...period, offset: period.offset + 1 })}>
          <ChevronRight size={20} />
        </button>
      </div>
    </div>
  )
}

export function Page({ title, children, action }: { title: string; children: ReactNode; action?: ReactNode }) {
  return (
    <div className="page">
      <header className="page-header">
        <h1>{title}</h1>
        {action}
      </header>
      {children}
    </div>
  )
}

export function PremiumLock({ text }: { text: string }) {
  const nav = useNavigate()
  return (
    <button className="premium-lock" onClick={() => nav('/premium')}>
      <Crown size={18} />
      <span>{text}</span>
    </button>
  )
}

export interface DonutSlice {
  value: number
  color: string
}

// Donut oddiy SVG halqa: har bir bo'lak stroke-dasharray bilan chiziladi.
export function Donut({ slices, size = 200, thickness = 26, children }: {
  slices: DonutSlice[]
  size?: number
  thickness?: number
  children?: ReactNode
}) {
  const r = (size - thickness) / 2
  const c = 2 * Math.PI * r
  const total = slices.reduce((s, x) => s + x.value, 0)
  const gap = slices.length > 1 ? 2 : 0
  let offset = 0
  return (
    <div className="donut" style={{ width: size, height: size }}>
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="var(--track)" strokeWidth={thickness} />
        {total > 0 && slices.map((s, i) => {
          const len = (s.value / total) * c
          const el = (
            <circle
              key={i}
              cx={size / 2}
              cy={size / 2}
              r={r}
              fill="none"
              stroke={s.color}
              strokeWidth={thickness}
              strokeDasharray={`${Math.max(len - gap, 0.5)} ${c}`}
              strokeDashoffset={-offset}
              transform={`rotate(-90 ${size / 2} ${size / 2})`}
            />
          )
          offset += len
          return el
        })}
      </svg>
      <div className="donut-center">{children}</div>
    </div>
  )
}

// MoneyInput summani yozish paytida guruhlab ko'rsatadi ("1 250 000").
// value/onChange xom qiymat bilan ishlaydi ("1250000"), shuning uchun parseMoney'ga to'g'ridan-to'g'ri beriladi.
export function MoneyInput({ value, onChange, decimals = 2, allowNegative = false, ...rest }: {
  value: string
  onChange: (raw: string) => void
  decimals?: number
  allowNegative?: boolean
  placeholder?: string
  disabled?: boolean
}) {
  const ref = useRef<HTMLInputElement>(null)
  // Formatlashdan keyin kursor joyini tiklash uchun: kursordan oldingi belgilar soni (bo'shliqsiz)
  const caret = useRef<number | null>(null)

  useLayoutEffect(() => {
    const el = ref.current
    if (!el || caret.current === null) return
    let seen = 0
    let pos = 0
    while (pos < el.value.length && seen < caret.current) {
      if (el.value[pos] !== ' ') seen++
      pos++
    }
    el.setSelectionRange(pos, pos)
    caret.current = null
  })

  return (
    <input
      {...rest}
      ref={ref}
      inputMode="decimal"
      autoComplete="off"
      value={groupDigits(value)}
      onChange={(e) => {
        const raw = cleanAmount(e.target.value, decimals, allowNegative)
        if (raw === null) return
        const before = e.target.value.slice(0, e.target.selectionStart ?? e.target.value.length)
        caret.current = before.replace(/\s/g, '').length - (e.target.value.replace(/\s/g, '').length - raw.length)
        onChange(raw)
      }}
    />
  )
}
