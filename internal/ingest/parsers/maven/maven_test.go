package maven

import (
	"testing"

	"github.com/nesbite/atlas/internal/ingest/parsers/depmodel"
)

func TestParsePomXML_dependencies_parsed_with_names_and_versions(t *testing.T) {
	content := []byte(`<project>
		<dependencies>
			<dependency>
				<groupId>com.google.guava</groupId>
				<artifactId>guava</artifactId>
				<version>31.1-jre</version>
			</dependency>
			<dependency>
				<groupId>org.apache.commons</groupId>
				<artifactId>commons-lang3</artifactId>
				<version>3.12.0</version>
			</dependency>
		</dependencies>
	</project>`)

	got := ParsePomXML(content, "pom.xml")

	if len(got) != 2 {
		t.Fatalf("expected 2 parsed deps, got %d: %+v", len(got), got)
	}

	byName := make(map[string]depmodel.ParsedDep)
	for _, d := range got {
		byName[d.Name] = d
	}

	guava, ok := byName["com.google.guava:guava"]
	if !ok {
		t.Fatal(`expected "com.google.guava:guava" in parsed deps`)
	}
	if guava.Version != "31.1-jre" || guava.SourceFile != "pom.xml" {
		t.Errorf("unexpected guava fields: %+v", guava)
	}

	commons, ok := byName["org.apache.commons:commons-lang3"]
	if !ok {
		t.Fatal(`expected "org.apache.commons:commons-lang3" in parsed deps`)
	}
	if commons.Version != "3.12.0" {
		t.Errorf("Version = %q, want %q", commons.Version, "3.12.0")
	}
}

func TestParsePomXML_scope_to_dep_type_mapping(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  string
	}{
		{"compile default when scope omitted", "", depmodel.Direct},
		{"explicit compile scope", "compile", depmodel.Direct},
		{"test scope maps to dev", "test", depmodel.Dev},
		{"provided scope maps to direct", "provided", depmodel.Direct},
		{"runtime scope maps to direct", "runtime", depmodel.Direct},
		{"system scope maps to direct", "system", depmodel.Direct},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scopeXML := ""
			if tt.scope != "" {
				scopeXML = "<scope>" + tt.scope + "</scope>"
			}
			content := []byte(`<project><dependencies><dependency>` +
				`<groupId>acme</groupId><artifactId>widget</artifactId><version>1.0.0</version>` +
				scopeXML +
				`</dependency></dependencies></project>`)

			got := ParsePomXML(content, "pom.xml")
			if len(got) != 1 {
				t.Fatalf("expected 1 parsed dep, got %d", len(got))
			}
			if got[0].DepType != tt.want {
				t.Errorf("DepType = %q, want %q", got[0].DepType, tt.want)
			}
		})
	}
}

func TestParsePomXML_property_placeholder_resolved_from_same_file(t *testing.T) {
	content := []byte(`<project>
		<properties>
			<junit.version>4.13.2</junit.version>
		</properties>
		<dependencies>
			<dependency>
				<groupId>junit</groupId>
				<artifactId>junit</artifactId>
				<version>${junit.version}</version>
				<scope>test</scope>
			</dependency>
		</dependencies>
	</project>`)

	got := ParsePomXML(content, "pom.xml")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d", len(got))
	}
	if got[0].Version != "4.13.2" {
		t.Errorf("Version = %q, want %q (resolved from <properties>)", got[0].Version, "4.13.2")
	}
}

func TestParsePomXML_unresolvable_property_kept_as_raw_placeholder(t *testing.T) {
	// No <properties> section defines "parent.version" — this simulates a
	// property inherited from a parent POM, which this parser never
	// fetches. The raw "${...}" string must be kept as-is.
	content := []byte(`<project>
		<dependencies>
			<dependency>
				<groupId>acme</groupId>
				<artifactId>widget</artifactId>
				<version>${parent.version}</version>
			</dependency>
		</dependencies>
	</project>`)

	got := ParsePomXML(content, "pom.xml")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep, got %d", len(got))
	}
	if got[0].Version != "${parent.version}" {
		t.Errorf("Version = %q, want raw placeholder %q", got[0].Version, "${parent.version}")
	}
}

func TestParsePomXML_skips_dependency_management_section(t *testing.T) {
	content := []byte(`<project>
		<dependencyManagement>
			<dependencies>
				<dependency>
					<groupId>org.springframework</groupId>
					<artifactId>spring-core</artifactId>
					<version>5.3.20</version>
				</dependency>
			</dependencies>
		</dependencyManagement>
		<dependencies>
			<dependency>
				<groupId>com.google.guava</groupId>
				<artifactId>guava</artifactId>
				<version>31.1-jre</version>
			</dependency>
		</dependencies>
	</project>`)

	got := ParsePomXML(content, "pom.xml")

	if len(got) != 1 {
		t.Fatalf("expected 1 parsed dep (dependencyManagement skipped), got %d: %+v", len(got), got)
	}
	if got[0].Name != "com.google.guava:guava" {
		t.Errorf("Name = %q, want %q", got[0].Name, "com.google.guava:guava")
	}
}

func TestParsePomXML_invalid_xml_returns_empty(t *testing.T) {
	got := ParsePomXML([]byte(`<project><dependencies>`), "pom.xml")
	if len(got) != 0 {
		t.Errorf("expected empty slice for malformed XML, got %d items", len(got))
	}
}

func TestParsePomXML_missing_dependencies_returns_empty(t *testing.T) {
	content := []byte(`<project><artifactId>acme-lib</artifactId></project>`)
	got := ParsePomXML(content, "pom.xml")
	if len(got) != 0 {
		t.Errorf("expected empty slice for manifest with no dependencies, got %d items", len(got))
	}
}

func TestParsePomXML_empty_groupid_or_artifactid_skipped(t *testing.T) {
	content := []byte(`<project>
		<dependencies>
			<dependency>
				<artifactId>no-group</artifactId>
				<version>1.0.0</version>
			</dependency>
			<dependency>
				<groupId>no-artifact</groupId>
				<version>1.0.0</version>
			</dependency>
			<dependency>
				<groupId>acme</groupId>
				<artifactId>widget</artifactId>
				<version>1.0.0</version>
			</dependency>
		</dependencies>
	</project>`)

	got := ParsePomXML(content, "pom.xml")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep (missing groupId/artifactId skipped), got %d: %+v", len(got), got)
	}
	if got[0].Name != "acme:widget" {
		t.Errorf("Name = %q, want %q", got[0].Name, "acme:widget")
	}
}

func TestParsePomXML_ecosystem_always_maven(t *testing.T) {
	content := []byte(`<project><dependencies><dependency>
		<groupId>acme</groupId><artifactId>widget</artifactId><version>1.0.0</version>
	</dependency></dependencies></project>`)
	got := ParsePomXML(content, "pom.xml")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].Ecosystem != depmodel.EcosystemMaven {
		t.Errorf("Ecosystem = %q, want %q", got[0].Ecosystem, depmodel.EcosystemMaven)
	}
}

func TestParsePomXML_source_file_path_set(t *testing.T) {
	content := []byte(`<project><dependencies><dependency>
		<groupId>acme</groupId><artifactId>widget</artifactId><version>1.0.0</version>
	</dependency></dependencies></project>`)
	got := ParsePomXML(content, "modules/billing/pom.xml")
	if len(got) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(got))
	}
	if got[0].SourceFile != "modules/billing/pom.xml" {
		t.Errorf("SourceFile = %q, want %q", got[0].SourceFile, "modules/billing/pom.xml")
	}
}
