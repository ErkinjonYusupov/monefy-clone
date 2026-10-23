-- Premium eslatmalari: qaysi muddat (premium_until qiymati) uchun xabar yuborilgani.
-- Muddat uzaytirilsa premium_until o'zgaradi va yangi muddat uchun eslatma qayta yuboriladi.
ALTER TABLE users
    ADD COLUMN premium_reminded_for TIMESTAMPTZ,  -- "3 kun qoldi" yuborilgan muddat
    ADD COLUMN premium_expired_for  TIMESTAMPTZ;  -- "tugadi" yuborilgan muddat

CREATE INDEX IF NOT EXISTS users_premium_until_idx ON users(premium_until) WHERE premium_until IS NOT NULL;
