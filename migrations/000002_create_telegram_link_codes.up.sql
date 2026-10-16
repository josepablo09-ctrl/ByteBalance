-- Códigos de un solo uso para vincular una cuenta con un chat de Telegram.
-- Solo se guarda el SHA-256 del código; el código en claro nunca se persiste.
CREATE TABLE telegram_link_codes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash  BYTEA       NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT telegram_link_codes_code_hash_key UNIQUE (code_hash),
    CONSTRAINT telegram_link_codes_hash_length CHECK (octet_length(code_hash) = 32)
);

CREATE INDEX telegram_link_codes_user_id_idx ON telegram_link_codes (user_id);

CREATE TRIGGER telegram_link_codes_set_updated_at
    BEFORE UPDATE ON telegram_link_codes
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
