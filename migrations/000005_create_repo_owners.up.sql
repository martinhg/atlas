CREATE TABLE IF NOT EXISTS repo_owners (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id      UUID         NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    pattern      VARCHAR(500) NOT NULL,
    owner        VARCHAR(255) NOT NULL,
    owner_type   VARCHAR(20)  NOT NULL,
    source       VARCHAR(20)  NOT NULL DEFAULT 'codeowners',
    line_number  INTEGER,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_owner_type CHECK (owner_type IN ('team', 'user', 'email')),
    UNIQUE (repo_id, pattern, owner)
);

CREATE INDEX IF NOT EXISTS idx_repo_owners_repo_id ON repo_owners (repo_id);
