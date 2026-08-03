package gemfile

import (
	"testing"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

func TestParseGemfile_direct_dep_no_version(t *testing.T) {
	content := []byte(`gem 'pg'`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "pg" {
		t.Errorf("Name = %q, want %q", got[0].Name, "pg")
	}
	if got[0].Version != "" {
		t.Errorf("Version = %q, want empty", got[0].Version)
	}
	if got[0].DepType != depmodel.Direct {
		t.Errorf("DepType = %q, want %q", got[0].DepType, depmodel.Direct)
	}
}

func TestParseGemfile_direct_dep_with_version(t *testing.T) {
	content := []byte(`gem 'rails', '~> 7.0'`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "rails" {
		t.Errorf("Name = %q, want %q", got[0].Name, "rails")
	}
	if got[0].Version != "~> 7.0" {
		t.Errorf("Version = %q, want %q", got[0].Version, "~> 7.0")
	}
}

func TestParseGemfile_multiple_version_constraints_uses_first(t *testing.T) {
	content := []byte(`gem 'puma', '>= 5.0', '< 6.0'`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Version != ">= 5.0" {
		t.Errorf("Version = %q, want %q (first version constraint)", got[0].Version, ">= 5.0")
	}
}

func TestParseGemfile_double_quoted_strings(t *testing.T) {
	content := []byte(`gem "sidekiq", "~> 7.1"`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "sidekiq" || got[0].Version != "~> 7.1" {
		t.Errorf("unexpected parsed dep: %+v", got[0])
	}
}

func TestParseGemfile_group_test_and_development_marks_dev(t *testing.T) {
	content := []byte(`
gem 'rails'

group :development, :test do
  gem 'rspec-rails'
  gem 'pry'
end

gem 'pg'
`)

	got := ParseGemfile(content, "Gemfile")

	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}

	if len(got) != 4 {
		t.Fatalf("expected 4 parsed deps, got %d: %+v", len(got), got)
	}
	if byName["rails"].DepType != depmodel.Direct {
		t.Errorf("rails DepType = %q, want %q", byName["rails"].DepType, depmodel.Direct)
	}
	if byName["pg"].DepType != depmodel.Direct {
		t.Errorf("pg DepType = %q, want %q", byName["pg"].DepType, depmodel.Direct)
	}
	if byName["rspec-rails"].DepType != depmodel.Dev {
		t.Errorf("rspec-rails DepType = %q, want %q", byName["rspec-rails"].DepType, depmodel.Dev)
	}
	if byName["pry"].DepType != depmodel.Dev {
		t.Errorf("pry DepType = %q, want %q", byName["pry"].DepType, depmodel.Dev)
	}
}

func TestParseGemfile_group_outside_marks_direct(t *testing.T) {
	content := []byte(`
group :production do
  gem 'pg'
end
`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].DepType != depmodel.Direct {
		t.Errorf("DepType = %q, want %q (group :production is not dev)", got[0].DepType, depmodel.Direct)
	}
}

func TestParseGemfile_git_option_empty_version(t *testing.T) {
	content := []byte(`gem 'sidekiq', git: 'https://github.com/sidekiq/sidekiq.git'`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "sidekiq" {
		t.Errorf("Name = %q, want %q", got[0].Name, "sidekiq")
	}
	if got[0].Version != "" {
		t.Errorf("Version = %q, want empty (git source has no version)", got[0].Version)
	}
}

func TestParseGemfile_path_option_empty_version(t *testing.T) {
	content := []byte(`gem 'local_gem', path: '../local_gem'`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "local_gem" {
		t.Errorf("Name = %q, want %q", got[0].Name, "local_gem")
	}
	if got[0].Version != "" {
		t.Errorf("Version = %q, want empty (path source has no version)", got[0].Version)
	}
}

func TestParseGemfile_github_option_empty_version(t *testing.T) {
	content := []byte(`gem 'httparty', github: 'jnunemaker/httparty'`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Version != "" {
		t.Errorf("Version = %q, want empty (github source has no version)", got[0].Version)
	}
}

func TestParseGemfile_skips_source_ruby_git_path_platforms_directives(t *testing.T) {
	content := []byte(`
source 'https://rubygems.org'
ruby '3.2.0'
git_source(:github) { |repo| "https://github.com/#{repo}.git" }

gem 'rails'
`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep (only rails), got %d: %+v", len(got), got)
	}
	if got[0].Name != "rails" {
		t.Errorf("Name = %q, want %q", got[0].Name, "rails")
	}
}

func TestParseGemfile_skips_comments_and_blank_lines(t *testing.T) {
	content := []byte(`
# This is a comment
gem 'rails' # inline comment

# another comment
gem 'pg'
`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 2 {
		t.Fatalf("expected 2 parsed deps, got %d: %+v", len(got), got)
	}
}

func TestParseGemfile_ecosystem_always_rubygems(t *testing.T) {
	got := ParseGemfile([]byte(`gem 'rails'`), "Gemfile")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Ecosystem != depmodel.EcosystemRubyGems {
		t.Errorf("Ecosystem = %q, want %q", got[0].Ecosystem, depmodel.EcosystemRubyGems)
	}
}

func TestParseGemfile_source_file_path_set(t *testing.T) {
	got := ParseGemfile([]byte(`gem 'rails'`), "services/api/Gemfile")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].SourceFile != "services/api/Gemfile" {
		t.Errorf("SourceFile = %q, want %q", got[0].SourceFile, "services/api/Gemfile")
	}
}

func TestParseGemfile_empty_content_returns_empty(t *testing.T) {
	got := ParseGemfile([]byte(""), "Gemfile")
	if len(got) != 0 {
		t.Errorf("expected empty slice for empty content, got %d items", len(got))
	}
}

func TestParseGemfile_nested_block_inside_group_inherits_dev(t *testing.T) {
	content := []byte(`
group :test do
  platforms :ruby do
    gem 'byebug'
  end
end
`)

	got := ParseGemfile(content, "Gemfile")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d: %+v", len(got), got)
	}
	if got[0].Name != "byebug" {
		t.Errorf("Name = %q, want %q", got[0].Name, "byebug")
	}
	if got[0].DepType != depmodel.Dev {
		t.Errorf("DepType = %q, want %q (nested block inside group :test)", got[0].DepType, depmodel.Dev)
	}
}

func TestParseGemfile_group_ends_restores_direct(t *testing.T) {
	content := []byte(`
group :test do
  gem 'rspec'
end
gem 'pg'
`)

	got := ParseGemfile(content, "Gemfile")

	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}
	if byName["rspec"].DepType != depmodel.Dev {
		t.Errorf("rspec DepType = %q, want %q", byName["rspec"].DepType, depmodel.Dev)
	}
	if byName["pg"].DepType != depmodel.Direct {
		t.Errorf("pg DepType = %q, want %q (after group end)", byName["pg"].DepType, depmodel.Direct)
	}
}
