CREATE TABLE IF NOT EXISTS vulnerabilities (
    id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    osv_id              VARCHAR(50)   NOT NULL,
    cve_id              VARCHAR(50),
    ecosystem           VARCHAR(50)   NOT NULL,
    package_name        VARCHAR(255)  NOT NULL,
    severity            VARCHAR(20)   NOT NULL DEFAULT 'unknown',
    cvss_score          NUMERIC(4,1),
    cvss_vector         TEXT,
    summary             TEXT,
    details             TEXT,
    published_at        TIMESTAMPTZ,
    modified_at         TIMESTAMPTZ,
    fixed_version       VARCHAR(100),
    introduced_version  VARCHAR(100),
    affected_ranges     JSONB,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (osv_id)
);

CREATE INDEX IF NOT EXISTS idx_vulnerabilities_ecosystem_package ON vulnerabilities (ecosystem, package_name);
CREATE INDEX IF NOT EXISTS idx_vulnerabilities_severity          ON vulnerabilities (severity);

CREATE TABLE IF NOT EXISTS dependency_vulnerabilities (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    dep_id     UUID        NOT NULL REFERENCES dependencies(id) ON DELETE CASCADE,
    vuln_id    UUID        NOT NULL REFERENCES vulnerabilities(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (dep_id, vuln_id)
);

CREATE INDEX IF NOT EXISTS idx_dep_vulns_dep_id  ON dependency_vulnerabilities (dep_id);
CREATE INDEX IF NOT EXISTS idx_dep_vulns_vuln_id ON dependency_vulnerabilities (vuln_id);
