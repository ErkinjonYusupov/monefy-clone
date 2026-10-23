// Telegram WebApp SDK (index.html dagi telegram-web-app.js) uchun minimal tiplar va yordamchilar.

interface TgButton {
  show(): void
  hide(): void
  onClick(cb: () => void): void
  offClick(cb: () => void): void
}

interface TgWebApp {
  initData: string
  colorScheme: 'light' | 'dark'
  ready(): void
  setHeaderColor?(color: string): void
  setBackgroundColor?(color: string): void
  setBottomBarColor?(color: string): void
  expand(): void
  disableVerticalSwipes?(): void
  isVersionAtLeast?(version: string): boolean
  requestContact?(cb: (shared: boolean, res?: { response?: string }) => void): void
  close(): void
  BackButton: TgButton
  HapticFeedback?: {
    impactOccurred(style: 'light' | 'medium' | 'heavy'): void
    notificationOccurred(type: 'error' | 'success' | 'warning'): void
  }
  showConfirm?(message: string, cb: (ok: boolean) => void): void
  showAlert?(message: string, cb?: () => void): void
}

declare global {
  interface Window {
    Telegram?: { WebApp: TgWebApp }
  }
}

export const tg: TgWebApp | undefined = window.Telegram?.WebApp

// index.css dagi --bg bilan bir xil: Telegram sarlavhasi va foni ilova bilan mos tursin.
const APP_BG = '#17212b'

export function initTelegram() {
  if (!tg) return
  tg.ready()
  tg.expand()
  // Pastga surilganda Mini App yopilmasin (Bot API 7.7+)
  tg.disableVerticalSwipes?.()
  tg.setHeaderColor?.(APP_BG)
  tg.setBackgroundColor?.(APP_BG)
  tg.setBottomBarColor?.(APP_BG)
}

export const initData = () => tg?.initData ?? ''

export function haptic(kind: 'light' | 'success' | 'error' = 'light') {
  const h = tg?.HapticFeedback
  if (!h) return
  if (kind === 'light') h.impactOccurred('light')
  else h.notificationOccurred(kind)
}

// Telegram ichida bo'lmaganda (brauzerda test) oddiy confirm/alert ishlatiladi.
export function confirmDialog(message: string): Promise<boolean> {
  return new Promise((resolve) => {
    if (tg?.showConfirm && tg.initData) tg.showConfirm(message, resolve)
    else resolve(window.confirm(message))
  })
}

export function alertDialog(message: string): Promise<void> {
  return new Promise((resolve) => {
    if (tg?.showAlert && tg.initData) tg.showAlert(message, () => resolve())
    else {
      window.alert(message)
      resolve()
    }
  })
}

// requestPhone Telegram'ning "raqamni ulashish" oynasini ochadi (Bot API 6.9+).
// response — Telegram imzolagan kontakt; server uni tekshiradi.
export function requestPhone(): Promise<{ shared: boolean; response?: string }> {
  return new Promise((resolve) => {
    if (!tg?.requestContact || !tg.isVersionAtLeast?.('6.9')) {
      resolve({ shared: false })
      return
    }
    tg.requestContact((shared, res) => resolve({ shared, response: res?.response }))
  })
}

export const canRequestPhone = () => Boolean(tg?.requestContact && tg.isVersionAtLeast?.('6.9'))
