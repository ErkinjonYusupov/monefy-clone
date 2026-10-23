import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './index.css'
import { initTelegram } from './lib/telegram'

initTelegram()

// Telegram ishga tushirish parametrlarini URL hash'iga yozadi (#tgWebAppData=...).
// telegram-web-app.js ularni allaqachon o'qib bo'lgan; hash marshrut uchun ishlatiladi, shuning uchun
// tozalanadi. Aks holda bosh sahifa "/" emas deb hisoblanib, Telegram'da "Orqaga" tugmasi chiqardi.
if (!location.hash.startsWith('#/')) {
  history.replaceState(null, '', location.pathname + location.search + '#/')
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
