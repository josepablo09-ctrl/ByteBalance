-- 000001_init_schema: esquema inicial de ByteBalance (usuarios, finanzas y hábitos).
-- gen_random_uuid() es nativo desde PostgreSQL 13; no requiere extensiones.

-- Mantiene updated_at sincronizado en cada UPDATE.
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ---------------------------------------------------------------------------
-- Usuarios
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email            VARCHAR(255) NOT NULL,
    password_hash    TEXT         NOT NULL,
    full_name        VARCHAR(150) NOT NULL,
    default_currency CHAR(3)      NOT NULL DEFAULT 'USD',
    timezone         VARCHAR(64)  NOT NULL DEFAULT 'UTC',
    telegram_chat_id BIGINT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ,
    CONSTRAINT users_email_not_blank CHECK (btrim(email) <> ''),
    CONSTRAINT users_default_currency_iso CHECK (default_currency ~ '^[A-Z]{3}$')
);

-- Email único sin distinguir mayúsculas, solo entre usuarios activos.
CREATE UNIQUE INDEX users_email_key ON users (lower(email)) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX users_telegram_chat_id_key ON users (telegram_chat_id)
    WHERE telegram_chat_id IS NOT NULL AND deleted_at IS NULL;

CREATE TRIGGER users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Finanzas: categorías y transacciones
-- ---------------------------------------------------------------------------
CREATE TABLE categories (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       VARCHAR(80) NOT NULL,
    type       VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT categories_type_check CHECK (type IN ('income', 'expense')),
    CONSTRAINT categories_name_not_blank CHECK (btrim(name) <> '')
);

CREATE UNIQUE INDEX categories_user_name_type_key ON categories (user_id, lower(name), type)
    WHERE deleted_at IS NULL;

CREATE TRIGGER categories_set_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE transactions (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID          NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category_id UUID          REFERENCES categories (id) ON DELETE SET NULL,
    type        VARCHAR(10)   NOT NULL,
    amount      NUMERIC(14,2) NOT NULL,
    currency    CHAR(3)       NOT NULL,
    description VARCHAR(255),
    occurred_at TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    CONSTRAINT transactions_type_check CHECK (type IN ('income', 'expense')),
    CONSTRAINT transactions_amount_positive CHECK (amount > 0),
    CONSTRAINT transactions_currency_iso CHECK (currency ~ '^[A-Z]{3}$')
);

-- Consulta principal: movimientos de un usuario ordenados por fecha.
CREATE INDEX transactions_user_occurred_at_idx ON transactions (user_id, occurred_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX transactions_category_id_idx ON transactions (category_id);

CREATE TRIGGER transactions_set_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Hábitos y su registro diario
-- ---------------------------------------------------------------------------
CREATE TABLE habits (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    description   TEXT,
    frequency     VARCHAR(10)  NOT NULL DEFAULT 'daily',
    target_count  INTEGER      NOT NULL DEFAULT 1,
    reminder_time TIME,
    is_active     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT habits_frequency_check CHECK (frequency IN ('daily', 'weekly', 'monthly')),
    CONSTRAINT habits_target_count_positive CHECK (target_count > 0),
    CONSTRAINT habits_name_not_blank CHECK (btrim(name) <> '')
);

CREATE INDEX habits_user_id_idx ON habits (user_id) WHERE deleted_at IS NULL;

CREATE TRIGGER habits_set_updated_at
    BEFORE UPDATE ON habits
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE habit_logs (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id   UUID        NOT NULL REFERENCES habits (id) ON DELETE CASCADE,
    logged_on  DATE        NOT NULL,
    count      INTEGER     NOT NULL DEFAULT 1,
    note       TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT habit_logs_count_positive CHECK (count > 0),
    -- Un registro por hábito y día; el índice también cubre consultas por rango de fechas.
    CONSTRAINT habit_logs_habit_logged_on_key UNIQUE (habit_id, logged_on)
);

CREATE TRIGGER habit_logs_set_updated_at
    BEFORE UPDATE ON habit_logs
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
