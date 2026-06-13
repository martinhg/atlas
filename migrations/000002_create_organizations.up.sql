CREATE TABLE IF NOT EXISTS organizations (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    github_id              BIGINT       NOT NULL UNIQUE,
    name                   VARCHAR(255) NOT NULL,
    slug                   VARCHAR(255) NOT NULL UNIQUE,
    github_installation_id BIGINT       UNIQUE,
    owner_id               UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_synced_at         TIMESTAMPTZ,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_organizations_slug     ON organizations (slug);
CREATE INDEX IF NOT EXISTS idx_organizations_owner_id ON organizations (owner_id);
