CREATE INDEX IF NOT EXISTS idx_repositories_name_pattern ON repositories (name text_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_repositories_full_name_pattern ON repositories (full_name text_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_dependencies_name_pattern ON dependencies (name text_pattern_ops);
