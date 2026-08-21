ALTER TABLE settlements
    ADD COLUMN IF NOT EXISTS reversal_reason text,
    ADD COLUMN IF NOT EXISTS correction_of_id text,
    ADD COLUMN IF NOT EXISTS idempotency_key text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ledger_version_at bigint NOT NULL DEFAULT 1;

CREATE UNIQUE INDEX IF NOT EXISTS idx_settlements_idempotency
    ON settlements(idempotency_key) WHERE idempotency_key <> '';

CREATE TABLE IF NOT EXISTS ledger_entries (
    id bigserial PRIMARY KEY,
    settlement_id text NOT NULL REFERENCES settlements(id),
    claim_id text NOT NULL REFERENCES expense_claims(id),
    kind text NOT NULL CHECK (kind IN ('occupy', 'release')),
    amount numeric(18,2) NOT NULL CHECK (amount > 0),
    balance numeric(18,2) NOT NULL CHECK (balance >= 0),
    occurred_at timestamptz NOT NULL,
    actor_id text NOT NULL,
    reason text NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_claim_time
    ON ledger_entries(claim_id, occurred_at, id);

