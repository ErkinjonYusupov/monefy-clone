import { useEffect, useState } from 'react'

// Ko'rilgan davrlar natijasi xotirada saqlanadi: "Oktabr → Sentabr → Oktabr" da qayta so'rov ketmaydi.
// Har qanday o'zgarishdan keyin store.refresh() keshni tozalaydi, shuning uchun eskirgan raqam ko'rinmaydi.
const cache = new Map<string, unknown>()

export function clearCache() {
  cache.clear()
}

// useCached key bo'yicha keshdan oladi, bo'lmasa fetcher'ni chaqiradi.
// Yangi ma'lumot kelguncha oldingisi ko'rinib turadi (ekran "sakramaydi").
// revision o'zgarsa (store.refresh), kesh tozalangan bo'ladi va qayta so'raladi.
export function useCached<T>(key: string, revision: number, fetcher: () => Promise<T>): T | undefined {
  const [data, setData] = useState<T | undefined>(() => cache.get(key) as T | undefined)

  useEffect(() => {
    const hit = cache.get(key) as T | undefined
    if (hit !== undefined) {
      setData(hit)
      return
    }
    // Tez-tez bosilganda eski javob yangisining ustiga yozilmasin
    let cancelled = false
    fetcher()
      .then((v) => {
        cache.set(key, v)
        if (!cancelled) setData(v)
      })
      .catch(console.error)
    return () => {
      cancelled = true
    }
    // fetcher har renderda yangi funksiya; so'rov faqat key/revision bo'yicha
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, revision])

  return data
}
