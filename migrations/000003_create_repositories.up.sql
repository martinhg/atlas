CREATE TABLE IF NOT EXISTS repositories (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID         NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    github_id      BIGINT       NOT NULL,
    name           VARCHAR(255) NOT NULL,
    full_name      VARCHAR(512) NOT NULL,
    description    TEXT,
    default_branch VARCHAR(255) NOT NULL DEFAULT 'main',
    language       VARCHAR(100),
    private        BOOLEAN      NOT NULL DEFAULT false,
    fork           BOOLEAN      NOT NULL DEFAULT false,
    stars          INTEGER      NOT NULL DEFAULT 0,
    last_synced_at TIMESTAMPTZ,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, github_id)
);

CREATE INDEX IF NOT EXISTS idx_repositories_org_id    ON repositories (org_id);
CREATE INDEX IF NOT EXISTS idx_repositories_full_name ON repositories (full_name);
