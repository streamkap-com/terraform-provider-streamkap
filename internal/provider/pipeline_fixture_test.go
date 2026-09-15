package provider

import (
	"regexp"
	"strings"
	"testing"
)

func TestPipelineReplicationNames(t *testing.T) {
	seen := map[string]bool{}
	for _, name := range []string{strings.Repeat("Fixture-Name-", 10) + "one", strings.Repeat("Fixture-Name-", 10) + "two"} {
		config := pipelineSrcPostgreSQLResourceDef(name)
		if config != pipelineSrcPostgreSQLResourceDef(name) {
			t.Fatal("replication names changed for the same fixture")
		}
		for _, field := range []string{"slot_name", "publication_name"} {
			match := regexp.MustCompile(field + `\s*=\s*"([^"]+)"`).FindStringSubmatch(config)
			if len(match) != 2 {
				t.Fatalf("missing %s", field)
			}
			value := match[1]
			if !regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`).MatchString(value) {
				t.Fatalf("invalid PostgreSQL identifier %q", value)
			}
			if seen[value] {
				t.Fatalf("replication identifier reused between fixtures: %s", value)
			}
			seen[value] = true
		}
	}
}
