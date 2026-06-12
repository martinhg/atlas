#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SKILLS_SRC="$REPO_ROOT/skills"

link_skills() {
  local tool="$1"
  local target="$2"
  mkdir -p "$target"

  for skill_dir in "$SKILLS_SRC"/*/; do
    skill_name="$(basename "$skill_dir")"
    link="$target/$skill_name"

    [ -L "$link" ] && rm "$link"
    ln -s "$skill_dir" "$link"
  done

  echo "[$tool] Linked $(ls -1 "$target" | wc -l | tr -d ' ') skills -> $target"
}

link_skills "claude" "$REPO_ROOT/.claude/skills"
link_skills "gemini" "$REPO_ROOT/.gemini/skills"
