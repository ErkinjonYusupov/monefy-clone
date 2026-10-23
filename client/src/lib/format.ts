// Summalar serverda tiyinda (×100) saqlanadi.

export function formatMoney(minor: number, currency?: string): string {
  const neg = minor < 0
  const abs = Math.abs(minor)
  const whole = Math.floor(abs / 100)
  const frac = abs % 100
  let s = whole.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ' ')
  if (frac) s += ',' + frac.toString().padStart(2, '0')
  if (neg) s = '−' + s
  return currency ? `${s} ${currencyLabel(currency)}` : s
}

export function currencyLabel(code: string): string {
  return code === 'UZS' ? "so'm" : code
}

// "1250.5" yoki "1250,5" -> 125050. Noto'g'ri bo'lsa null.
export function parseMoney(input: string): number | null {
  const s = input.replace(/\s/g, '').replace(',', '.')
  if (!/^\d+(\.\d{0,2})?$/.test(s)) return null
  const [w, f = ''] = s.split('.')
  const v = Number(w) * 100 + Number(f.padEnd(2, '0'))
  return Number.isSafeInteger(v) ? v : null
}

export function toInput(minor: number): string {
  const w = Math.floor(minor / 100)
  const f = minor % 100
  return f ? `${w}.${f.toString().padStart(2, '0')}` : String(w)
}

export const CURRENCIES = ['UZS', 'USD', 'EUR', 'RUB', 'KZT', 'KGS', 'TJS', 'TRY', 'CNY', 'GBP', 'AED']

const MONTHS = ['yanvar', 'fevral', 'mart', 'aprel', 'may', 'iyun', 'iyul', 'avgust', 'sentabr', 'oktabr', 'noyabr', 'dekabr']
const MONTHS_SHORT = ['yan', 'fev', 'mar', 'apr', 'may', 'iyn', 'iyl', 'avg', 'sen', 'okt', 'noy', 'dek']

export function ymd(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

export function parseYmd(s: string): Date {
  const [y, m, d] = s.split('-').map(Number)
  return new Date(y, m - 1, d)
}

export function humanDate(s: string): string {
  const d = parseYmd(s)
  const today = new Date()
  const yesterday = new Date()
  yesterday.setDate(today.getDate() - 1)
  if (ymd(d) === ymd(today)) return 'Bugun'
  if (ymd(d) === ymd(yesterday)) return 'Kecha'
  const base = `${d.getDate()}-${MONTHS[d.getMonth()]}`
  return d.getFullYear() === today.getFullYear() ? base : `${base}, ${d.getFullYear()}`
}

export type PeriodType = 'day' | 'week' | 'month' | 'year' | 'all'

export interface Period {
  type: PeriodType
  offset: number // 0 = joriy, -1 = oldingi
}

export function periodRange(p: Period): { from?: string; to?: string; label: string } {
  const now = new Date()
  switch (p.type) {
    case 'day': {
      const d = new Date(now.getFullYear(), now.getMonth(), now.getDate() + p.offset)
      return { from: ymd(d), to: ymd(d), label: humanDate(ymd(d)) }
    }
    case 'week': {
      const dow = (now.getDay() + 6) % 7 // dushanba = 0
      const start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - dow + p.offset * 7)
      const end = new Date(start.getFullYear(), start.getMonth(), start.getDate() + 6)
      const label = `${start.getDate()} ${MONTHS_SHORT[start.getMonth()]} – ${end.getDate()} ${MONTHS_SHORT[end.getMonth()]}`
      return { from: ymd(start), to: ymd(end), label }
    }
    case 'month': {
      const start = new Date(now.getFullYear(), now.getMonth() + p.offset, 1)
      const end = new Date(start.getFullYear(), start.getMonth() + 1, 0)
      const name = MONTHS[start.getMonth()]
      const label = start.getFullYear() === now.getFullYear()
        ? name[0].toUpperCase() + name.slice(1)
        : `${name[0].toUpperCase() + name.slice(1)} ${start.getFullYear()}`
      return { from: ymd(start), to: ymd(end), label }
    }
    case 'year': {
      const y = now.getFullYear() + p.offset
      return { from: `${y}-01-01`, to: `${y}-12-31`, label: String(y) }
    }
    case 'all':
      return { label: 'Butun davr' }
  }
}

export const PERIOD_LABELS: Record<PeriodType, string> = {
  day: 'Kun', week: 'Hafta', month: 'Oy', year: 'Yil', all: 'Hammasi',
}
