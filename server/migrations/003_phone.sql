-- Telefon raqami: foydalanuvchi Mini App'da o'z kontaktini ulashadi (Telegram tasdiqlagan).
ALTER TABLE users
    ADD COLUMN phone             TEXT NOT NULL DEFAULT '',
    ADD COLUMN phone_verified_at TIMESTAMPTZ;

-- Admin ro'yxati yangi foydalanuvchilardan boshlab saralanadi.
CREATE INDEX IF NOT EXISTS users_created_idx ON users(created_at DESC);
