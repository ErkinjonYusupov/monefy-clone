-- Hisob balansi account_id bo'yicha hisoblanadi; indekssiz butun jadval o'qilardi.
CREATE INDEX IF NOT EXISTS transactions_account_idx ON transactions(account_id);
-- Kategoriya o'chirishdan oldingi tekshiruv va FK uchun.
CREATE INDEX IF NOT EXISTS transactions_category_idx ON transactions(category_id);
CREATE INDEX IF NOT EXISTS transfers_from_account_idx ON transfers(from_account_id);
CREATE INDEX IF NOT EXISTS transfers_to_account_idx ON transfers(to_account_id);
