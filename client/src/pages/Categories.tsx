import { Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { CategoryBadge, Page, PremiumLock } from '../components/ui'
import { api, ApiError } from '../lib/api'
import { COLORS, Icon, ICON_NAMES } from '../lib/icons'
import { useApp } from '../lib/store'
import { alertDialog, confirmDialog, haptic } from '../lib/telegram'
import type { Kind } from '../lib/types'

export function Categories() {
  const { me, categories } = useApp()
  const nav = useNavigate()
  const [kind, setKind] = useState<Kind>('expense')

  return (
    <Page
      title="Kategoriyalar"
      action={me.is_premium && (
        <button className="icon-btn" onClick={() => nav(`/categories/new?kind=${kind}`)} aria-label="Qo'shish"><Plus size={22} /></button>
      )}
    >
      <div className="seg wide">
        <button className={kind === 'expense' ? 'active' : ''} onClick={() => setKind('expense')}>Xarajat</button>
        <button className={kind === 'income' ? 'active' : ''} onClick={() => setKind('income')}>Daromad</button>
      </div>
      {!me.is_premium && <PremiumLock text="Kategoriyalarni boshqarish — Premium" />}
      <ul className="list">
        {categories.filter((c) => c.kind === kind).map((c) => (
          <li key={c.id}>
            <button
              className={`list-row ${c.is_archived ? 'dim' : ''}`}
              disabled={!me.is_premium}
              onClick={() => nav(`/categories/${c.id}`)}
            >
              <CategoryBadge icon={c.icon} color={c.color} size={38} />
              <span className="grow">{c.name}{c.is_archived && <small className="muted"> · arxiv</small>}</span>
            </button>
          </li>
        ))}
      </ul>
    </Page>
  )
}

export function CategoryForm() {
  const { id } = useParams()
  const [search] = useSearchParams()
  const { categories, refresh } = useApp()
  const nav = useNavigate()
  const existing = categories.find((c) => c.id === Number(id))
  const kind = (existing?.kind ?? search.get('kind') ?? 'expense') as Kind

  const [name, setName] = useState(existing?.name ?? '')
  const [icon, setIcon] = useState(existing?.icon ?? 'star')
  const [color, setColor] = useState(existing?.color ?? COLORS[5])
  const [archived, setArchived] = useState(existing?.is_archived ?? false)

  const save = async () => {
    if (!name.trim()) return alertDialog('Nomini kiriting')
    try {
      if (existing) await api.updateCategory(existing.id, { name, icon, color, is_archived: archived })
      else await api.createCategory({ kind, name, icon, color })
      haptic('success')
      await refresh()
      nav(-1)
    } catch (e) {
      alertDialog(e instanceof Error ? e.message : String(e))
    }
  }

  const remove = async () => {
    if (!existing || !(await confirmDialog(`"${existing.name}" o'chirilsinmi?`))) return
    try {
      await api.deleteCategory(existing.id)
      await refresh()
      nav(-1)
    } catch (e) {
      if (e instanceof ApiError && e.code === 'category_in_use') {
        if (await confirmDialog("Bu kategoriyada yozuvlar bor. O'rniga arxivlansinmi?")) {
          await api.updateCategory(existing.id, { is_archived: true })
          await refresh()
          nav(-1)
        }
      } else {
        alertDialog(e instanceof Error ? e.message : String(e))
      }
    }
  }

  return (
    <Page
      title={existing ? 'Kategoriya' : kind === 'expense' ? 'Yangi xarajat kategoriyasi' : 'Yangi daromad kategoriyasi'}
      action={existing && <button className="icon-btn" onClick={remove} aria-label="O'chirish"><Trash2 size={20} /></button>}
    >
      <div className="preview"><CategoryBadge icon={icon} color={color} size={64} /></div>
      <label className="field">
        <span>Nomi</span>
        <input value={name} maxLength={64} onChange={(e) => setName(e.target.value)} />
      </label>
      <div className="field">
        <span>Rang</span>
        <div className="color-pick">
          {COLORS.map((c) => (
            <button key={c} className={c === color ? 'selected' : ''} style={{ background: c }} onClick={() => setColor(c)} aria-label={c} />
          ))}
        </div>
      </div>
      <div className="field">
        <span>Ikonka</span>
        <div className="icon-pick">
          {ICON_NAMES.map((i) => (
            <button key={i} className={i === icon ? 'selected' : ''} onClick={() => setIcon(i)}><Icon name={i} /></button>
          ))}
        </div>
      </div>
      {existing && (
        <label className="check">
          <input type="checkbox" checked={archived} onChange={(e) => setArchived(e.target.checked)} />
          Arxivlangan (yangi yozuvda ko'rinmaydi)
        </label>
      )}
      <button className="btn primary wide" onClick={save}>Saqlash</button>
    </Page>
  )
}
