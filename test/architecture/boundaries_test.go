package architecture_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestModuleDependencyBoundaries(t *testing.T) {
	checkModuleBoundaries(t, "../../internal/catalog/...", "/internal/crm")
	checkModuleBoundaries(t, "../../internal/crm/...", "/internal/catalog")
}

func checkModuleBoundaries(t *testing.T, pattern, forbiddenModule string) {
	t.Helper()
	command := exec.Command("go", "list", "-f", `{{.ImportPath}} -> {{join .Imports " "}}`, pattern)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go list %s failed: %v\n%s", pattern, err, output)
	}
	for _, line := range strings.Split(string(output), "\n") {
		packagePath, imports, found := strings.Cut(line, " -> ")
		if !found {
			continue
		}
		for _, imported := range strings.Fields(imports) {
			if strings.Contains(imported, forbiddenModule) {
				t.Errorf("cross-module import forbidden: %s", line)
			}
			if strings.HasSuffix(packagePath, "/domain") && (imported == "database/sql" || imported == "github.com/gin-gonic/gin" || strings.Contains(imported, "/infrastructure/")) {
				t.Errorf("domain imports forbidden package: %s", line)
			}
			if strings.HasSuffix(packagePath, "/application") && (imported == "github.com/gin-gonic/gin" || strings.Contains(imported, "/infrastructure/")) {
				t.Errorf("application imports forbidden package: %s", line)
			}
			if strings.HasSuffix(packagePath, "/interfaces/rest") && strings.Contains(imported, "/infrastructure/") {
				t.Errorf("REST imports infrastructure directly: %s", line)
			}
		}
	}
}
