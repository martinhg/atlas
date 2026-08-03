package scan

import (
	"encoding/xml"

	"github.com/nesbite/atlas/internal/ingest/parsers/maven"
)

// NewMavenScanner returns an EcosystemScanner that discovers and parses
// pom.xml files within a directory tree, recursively, skipping the build
// output directory (target/) and the Maven Wrapper directory (.mvn/).
func NewMavenScanner() WalkDirScanner {
	return NewWalkDirScanner(WalkConfig{
		Name:     "maven",
		FileName: "pom.xml",
		SkipDirs: []string{"target", ".mvn"},
		Parse:    maven.ParsePomXML,
		Validate: isValidPomXML,
	})
}

// isValidPomXML reports whether data is a well-formed XML document whose
// root element is <project>. A pom.xml with no <dependencies> element is
// still valid — it simply produces zero deps, which is not an error.
func isValidPomXML(data []byte) bool {
	var v struct {
		XMLName xml.Name `xml:"project"`
	}
	return xml.Unmarshal(data, &v) == nil
}
