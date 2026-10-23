# Monefy clone: Telegram Mini App

Xarajat va daromadlarni hisoblash uchun Telegram Mini App.
Frontend React (Vite + TypeScript), backend Go, ma'lumotlar bazasi PostgreSQL.

```
client/   React Mini App
server/   Go API va Telegram bot (bitta binary)
```

## Ishga tushirish (lokal)

PostgreSQL Docker'dagi `credit_local_db` konteynerida (`localhost:5432`, `monefy` bazasi).
Jadvallar server ishga tushganda avtomatik yaratiladi (`server/migrations`).

```bash
# 1. Backend
cp .env.example .env      # repo ildizida; BOT_TOKEN, ADMIN_CHAT_ID ni to'ldiring
cd server
go run ./cmd/api          # http://localhost:8090

# 2. Frontend
cd client
npm install
npm run dev               # http://localhost:5173 (/api -> :8090 proxy)
```

Brauzerda Telegram'siz sinash uchun `.env` da `DEV_TELEGRAM_ID` ni qo'ying.
**Prod'da bu qiymat bo'sh bo'lishi shart**, aks holda har kim shu foydalanuvchi sifatida kiradi.

Telegram ichida sinash: frontend'ni HTTPS tunnel orqali oching
(`cloudflared tunnel --url http://localhost:5173`), so'ng URL'ni @BotFather'da
Mini App manzili qilib qo'ying va `WEBAPP_URL` ga yozing.

## Deploy (Dokploy)

Repo ildizidagi `Dockerfile` React'ni build qiladi va Go binary bilan bitta image'ga yig'adi.
Natijada bitta servis va bitta domen: `/` da Mini App, `/api` da API. CORS kerak emas.

1. Dokploy'da **Application** yarating, manba: shu GitHub repo, Build Type: **Dockerfile** (yo'l: `Dockerfile`).
2. **Environment** ga quyidagilarni yozing:
   ```
   DATABASE_URL=postgres://USER:PASS@HOST:5432/monefy?sslmode=disable
   BOT_TOKEN=...
   ADMIN_CHAT_ID=...
   WEBAPP_URL=https://monefy.example.com
   ```
   `DEV_TELEGRAM_ID` ni **qo'ymang**.
3. **Domains** da domen qo'shing, container port **8080**, HTTPS yoqilgan bo'lsin.
4. @BotFather'da `/newapp` yoki Menu Button orqali Mini App URL'ini shu domenga qo'ying.

`DATABASE_URL` dagi baza (masalan `monefy`) mavjud bo'lmasa, server ishga tushganda uni o'zi yaratadi
(buning uchun DB foydalanuvchisida `CREATEDB` huquqi kerak; Dokploy'ning Postgres servisi
foydalanuvchisida bu huquq bor). Jadvallar migratsiyalar orqali yaratiladi.

Postgres Dokploy'ning o'zida bo'lsa, `HOST` uchun uning **Internal Host** qiymatini oling
(Postgres servisining sahifasida ko'rsatilgan).

Bot long polling bilan ishlaydi, shuning uchun bitta `BOT_TOKEN` bilan faqat bitta nusxa ishlashi kerak.
Lokal server ham shu tokenni ishlatsa, lokalda `BOT_POLLING=false` qiling.

## Muhit o'zgaruvchilari (.env)

| O'zgaruvchi | Vazifasi |
|---|---|
| `DATABASE_URL` | PostgreSQL ulanish manzili |
| `BOT_TOKEN` | @BotFather tokeni; initData imzosini tekshirish va bot uchun |
| `ADMIN_CHAT_ID` | Admin buyruqlari faqat shu chatda ishlaydi; yangi foydalanuvchilar va premium so'rovlari shu yerga keladi |
| `WEBAPP_URL` | `/start` dagi "Ilovani ochish" tugmasi manzili |
| `CORS_ORIGINS` | Frontend boshqa domenda bo'lsa |
| `DEV_TELEGRAM_ID` | Faqat lokal test |
| `BOT_POLLING` | `false` bo'lsa bot xabarlarni qabul qilmaydi |
| `STATIC_DIR` | React build papkasi (Docker image'da `/app/web`, avtomatik) |

## Premium

Monefy Pro'dagi kabi quyidagilar faqat premium uchun ochiq:
ko'p valyutali hisoblar va kurslar, kategoriyalarni boshqarish, CSV eksport.

Rejalar va narxlar `server/internal/service/premium.go` da.
Hozircha to'lov qo'lda: foydalanuvchi ilovada "Premium olish" ni bosadi, so'rov admin chatiga keladi,
admin to'lovni olgach buyruq yuboradi. Har bir obuna `subscriptions` jadvalida summasi bilan yoziladi.
Muddat uzaytirilganda yangi davr joriy muddat tugagan kundan boshlanadi.

Admin chatidagi buyruqlar:

```
/stats                               foydalanuvchilar va tushum
/user <telegram_id>                  foydalanuvchi va obunalar tarixi
/grant <telegram_id> year            reja narxi bilan premium berish
/grant <telegram_id> 30d 0           ixtiyoriy muddat, bepul
/revoke <telegram_id>                premiumni bekor qilish
```

Click ulanganda `subscriptions.source = 'click'` va `provider_txn_id` (UNIQUE) ishlatiladi.
Callback handler `store.GrantSubscription` ni chaqiradi.

## Pul summalari

Barcha summalar `BIGINT` turida, tiyinda (×100) saqlanadi. API ham tiyinda qabul qiladi va qaytaradi.
