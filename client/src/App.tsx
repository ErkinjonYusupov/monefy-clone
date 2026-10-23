import { useEffect } from 'react'
import { HashRouter, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { AccountForm, Accounts, TransferForm } from './pages/Accounts'
import { Categories, CategoryForm } from './pages/Categories'
import Home from './pages/Home'
import { Menu, Rates } from './pages/Menu'
import Premium from './pages/Premium'
import Transactions from './pages/Transactions'
import TxForm from './pages/TxForm'
import { AppProvider } from './lib/store'
import { tg } from './lib/telegram'

// Bosh sahifadan boshqa joyda Telegram'ning "Orqaga" tugmasi ko'rinadi.
function BackButton() {
  const location = useLocation()
  const nav = useNavigate()
  useEffect(() => {
    // scroll konteyneri #root (index.css), yangi sahifa tepadan ochilsin
    document.getElementById('root')?.scrollTo(0, 0)
    if (!tg) return
    const back = () => nav(-1)
    if (location.pathname === '/') tg.BackButton.hide()
    else tg.BackButton.show()
    tg.BackButton.onClick(back)
    return () => tg?.BackButton.offClick(back)
  }, [location.pathname, nav])
  return null
}

export default function App() {
  return (
    <AppProvider>
      <HashRouter>
        <BackButton />
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/add/:kind" element={<TxForm />} />
          <Route path="/edit/:id" element={<TxForm />} />
          <Route path="/transactions" element={<Transactions />} />
          <Route path="/accounts" element={<Accounts />} />
          <Route path="/accounts/new" element={<AccountForm />} />
          <Route path="/accounts/:id" element={<AccountForm />} />
          <Route path="/transfer" element={<TransferForm />} />
          <Route path="/categories" element={<Categories />} />
          <Route path="/categories/new" element={<CategoryForm />} />
          <Route path="/categories/:id" element={<CategoryForm />} />
          <Route path="/rates" element={<Rates />} />
          <Route path="/menu" element={<Menu />} />
          <Route path="/premium" element={<Premium />} />
          <Route path="*" element={<Home />} />
        </Routes>
      </HashRouter>
    </AppProvider>
  )
}
