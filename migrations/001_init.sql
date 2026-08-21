CREATE TABLE IF NOT EXISTS grant_projects (
    id text PRIMARY KEY,
    name text NOT NULL,
    year integer NOT NULL,
    annual_limit numeric(18,2) NOT NULL CHECK (annual_limit >= 0),
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS claimants (
    id text PRIMARY KEY,
    display_name text NOT NULL,
    identity_digest text NOT NULL UNIQUE,
    plan_code text NOT NULL,
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS rule_versions (
    id text PRIMARY KEY,
    project_id text NOT NULL REFERENCES grant_projects(id),
    version integer NOT NULL,
    effective_from date NOT NULL,
    effective_to date,
    cap numeric(18,2) NOT NULL,
    segments jsonb NOT NULL,
    conditions jsonb NOT NULL,
    published_at timestamptz,
    UNIQUE(project_id, version)
);

CREATE TABLE IF NOT EXISTS expense_claims (
    id text PRIMARY KEY,
    claimant_id text NOT NULL REFERENCES claimants(id),
    project_id text NOT NULL REFERENCES grant_projects(id),
    category text NOT NULL,
    receipt_digest text NOT NULL,
    occurred_on date NOT NULL,
    amount numeric(18,2) NOT NULL CHECK (amount > 0),
    status text NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    idempotency_key text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL,
    UNIQUE(claimant_id, project_id, receipt_digest)
);

CREATE TABLE IF NOT EXISTS settlements (
    id text PRIMARY KEY,
    claim_id text NOT NULL REFERENCES expense_claims(id),
    rule_version_id text NOT NULL REFERENCES rule_versions(id),
    approved_amount numeric(18,2) NOT NULL,
    status text NOT NULL,
    explanation jsonb NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    confirmed_at timestamptz,
    UNIQUE(claim_id)
);

CREATE TABLE IF NOT EXISTS annual_ledgers (
    claimant_id text NOT NULL REFERENCES claimants(id),
    project_id text NOT NULL REFERENCES grant_projects(id),
    year integer NOT NULL,
    occupied numeric(18,2) NOT NULL DEFAULT 0,
    version bigint NOT NULL DEFAULT 1,
    PRIMARY KEY(claimant_id, project_id, year)
);

CREATE TABLE IF NOT EXISTS review_actions (
    id text PRIMARY KEY,
    claim_id text NOT NULL REFERENCES expense_claims(id),
    actor_id text NOT NULL,
    action text NOT NULL,
    reason text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_events (
    id bigserial PRIMARY KEY,
    request_id text NOT NULL,
    actor_id text NOT NULL,
    action text NOT NULL,
    subject_type text NOT NULL,
    subject_id text NOT NULL,
    detail jsonb NOT NULL,
    created_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_claims_review_queue ON expense_claims(status, occurred_on, id);
CREATE INDEX IF NOT EXISTS idx_rules_effective ON rule_versions(project_id, effective_from, effective_to);
CREATE INDEX IF NOT EXISTS idx_audit_subject ON audit_events(subject_type, subject_id, created_at);

