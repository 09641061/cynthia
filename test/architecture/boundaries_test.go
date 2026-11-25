package architecture_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestModuleDependencyBoundaries(t *testing.T) {
	checkModuleBoundaries(t, "../../internal/catalog/...", []string{"/internal/crm", "/internal/organizations", "/internal/calendar"})
	checkModuleBoundaries(t, "../../internal/crm/...", []string{"/internal/catalog", "/internal/organizations", "/internal/calendar"})
	checkModuleBoundaries(t, "../../internal/organizations/...", []string{"/internal/catalog", "/internal/crm", "/internal/calendar"})
	checkModuleBoundaries(t, "../../internal/calendar/...", []string{"/internal/catalog", "/internal/crm", "/internal/organizations"})
}

func checkModuleBoundaries(t *testing.T, pattern string, forbiddenModules []string) {
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
			for _, forbidden := range forbiddenModules {
				if !strings.Contains(imported, forbidden) {
					continue
				}
				isModuleOrACL := strings.HasSuffix(packagePath, "/infrastructure/acl") ||
					packagePath == "go-service/internal/calendar"
				isPublishedRoot := strings.HasSuffix(imported, forbidden)
				if !(isModuleOrACL && isPublishedRoot) {
					t.Errorf("cross-module import forbidden: %s", line)
				}
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
