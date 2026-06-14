CREATE TABLE IF NOT EXISTS dependencies (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    ecosystem  VARCHAR(50)  NOT NULL,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (ecosystem, name)
);

CREATE TABLE IF NOT EXISTS repo_dependencies (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    repo_id     UUID         NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    dep_id      UUID         NOT NULL REFERENCES dependencies(id) ON DELETE CASCADE,
    version     VARCHAR(100) NOT NULL,
    dep_type    VARCHAR(20)  NOT NULL,
    source_file VARCHAR(500) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (repo_id, dep_id, source_file)
);

CREATE INDEX IF NOT EXISTS idx_repo_dependencies_dep_id  ON repo_dependencies (dep_id);
CREATE INDEX IF NOT EXISTS idx_repo_dependencies_repo_id ON repo_dependencies (repo_id);
