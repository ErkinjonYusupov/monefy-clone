-- Barcha pul summalari BIGINT, eng kichik birlikda (summa * 100) saqlanadi.

CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    telegram_id     BIGINT      NOT NULL UNIQUE,
    first_name      TEXT        NOT NULL DEFAULT '',
    last_name       TEXT        NOT NULL DEFAULT '',
    username        TEXT        NOT NULL DEFAULT '',
    language_code   TEXT        NOT NULL DEFAULT '',
    base_currency   CHAR(3)     NOT NULL DEFAULT 'UZS',
    budget_amount   BIGINT,                         -- NULL = budget rejimi o'chiq
    premium_until   TIMESTAMPTZ,                    -- subscriptions jadvalidan hisoblanadi
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE accounts (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT        NOT NULL,
    currency        CHAR(3)     NOT NULL,
    initial_balance BIGINT      NOT NULL DEFAULT 0,
    icon            TEXT        NOT NULL DEFAULT 'wallet',
    is_archived     BOOLEAN     NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX accounts_user_idx ON accounts(user_id);

CREATE TABLE categories (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind            TEXT        NOT NULL CHECK (kind IN ('expense', 'income')),
    name            TEXT        NOT NULL,
    icon            TEXT        NOT NULL,
    color           TEXT        NOT NULL,
    sort_order      INT         NOT NULL DEFAULT 0,
    is_archived     BOOLEAN     NOT NULL DEFAULT false
);
CREATE INDEX categories_user_idx ON categories(user_id);

CREATE TABLE transactions (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id      BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    category_id     BIGINT      NOT NULL REFERENCES categories(id),
    amount          BIGINT      NOT NULL CHECK (amount > 0),
    note            TEXT        NOT NULL DEFAULT '',
    occurred_on     DATE        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX transactions_user_date_idx ON transactions(user_id, occurred_on);

CREATE TABLE transfers (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_account_id BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    to_account_id   BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    amount_from     BIGINT      NOT NULL CHECK (amount_from > 0),
    amount_to       BIGINT      NOT NULL CHECK (amount_to > 0),
    note            TEXT        NOT NULL DEFAULT '',
    occurred_on     DATE        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (from_account_id <> to_account_id)
);
CREATE INDEX transfers_user_date_idx ON transfers(user_id, occurred_on);

-- 1 birlik `currency` = `rate` birlik asosiy valyuta (premium).
CREATE TABLE currency_rates (
    user_id         BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency        CHAR(3)       NOT NULL,
    rate            NUMERIC(20,8) NOT NULL CHECK (rate > 0),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, currency)
);

-- Premium obunalar tarixi. Har bir yozuv muddatni uzaytiradi.
CREATE TABLE subscriptions (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan            TEXT        NOT NULL,           -- month | year | lifetime | custom
    source          TEXT        NOT NULL,           -- admin | click
    amount          BIGINT      NOT NULL DEFAULT 0, -- to'langan summa (tiyin)
    currency        CHAR(3)     NOT NULL DEFAULT 'UZS',
    provider_txn_id TEXT        UNIQUE,             -- Click tranzaksiya ID (keyinchalik)
    granted_by      BIGINT,                         -- admin telegram_id
    starts_at       TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    note            TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX subscriptions_user_idx ON subscriptions(user_id);
