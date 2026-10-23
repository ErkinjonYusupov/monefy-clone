import { Phone } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../lib/api'
import { canRequestPhone, haptic, requestPhone } from '../lib/telegram'

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

// PhoneGate foydalanuvchi telefon raqamini ulashmaguncha ilovani yopib turadi.
// Ochilishi bilan Telegram'ning "raqamni ulashish" oynasi chiqadi.
export default function PhoneGate({ onDone }: { onDone: () => Promise<void> }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const asked = useRef(false)

  const ask = useCallback(async () => {
    setBusy(true)
    setError(null)
    try {
      const { shared, response } = await requestPhone()
      if (!shared) {
        setError("Ilovadan foydalanish uchun telefon raqamingizni ulashing.")
        return
      }
      if (response) {
        await api.savePhone(response)
      } else {
        // Ba'zi klientlar imzolangan javob bermaydi; kontakt botga ham keladi va bot uni saqlaydi.
        let saved = false
        for (let i = 0; i < 8 && !saved; i++) {
          await sleep(1000)
          saved = !(await api.me()).phone_required
        }
        if (!saved) throw new Error("Raqam tasdiqlanmadi. Qayta urinib ko'ring.")
      }
      haptic('success')
      await onDone()
    } catch (e) {
      haptic('error')
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }, [onDone])

  useEffect(() => {
    // StrictMode effektni ikki marta chaqiradi; oyna faqat bir marta ochilsin
    if (asked.current || !canRequestPhone()) return
    asked.current = true
    ask()
  }, [ask])

  return (
    <div className="center-screen phone-gate">
      <span className="phone-icon"><Phone size={36} /></span>
      <h2>Telefon raqamingizni tasdiqlang</h2>
      <p className="muted">
        Monefy'dan foydalanish uchun Telegram'dagi telefon raqamingizni ulashing.
        Raqam faqat akkauntingizni aniqlash uchun ishlatiladi.
      </p>
      {canRequestPhone() ? (
        <button className="btn primary wide" disabled={busy} onClick={ask}>
          {busy ? 'Kutilmoqda…' : 'Raqamni ulashish'}
        </button>
      ) : (
        <p className="warning">Telegram ilovangizni yangilang: bu versiyada raqam ulashib bo'lmaydi.</p>
      )}
      {error && <p className="neg small">{error}</p>}
    </div>
  )
}
