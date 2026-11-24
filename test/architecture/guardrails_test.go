package architecture_test

import (
	"bytes"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestGoFilesMeetLineLimit(t *testing.T) {
	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := bytes.Count(content, []byte{'\n'})
		if len(content) > 0 && content[len(content)-1] != '\n' {
			lines++
		}
		if lines > 150 {
			t.Errorf("%s has %d lines; maximum is 150", path, lines)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGoFilesAreFormatted(t *testing.T) {
	root := repositoryRoot(t)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		formatted, err := format.Source(source)
		if err != nil {
			return err
		}
		if !bytes.Equal(source, formatted) {
			t.Errorf("%s is not gofmt-formatted", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnvironmentSecretsAreNotTracked(t *testing.T) {
	root := repositoryRoot(t)
	command := exec.Command("git", "ls-files", "-z")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	for _, path := range strings.Split(string(output), "\x00") {
		name := filepath.Base(path)
		if name == ".env" || (strings.HasPrefix(name, ".env.") && name != ".env.example") {
			t.Errorf("environment secret file is tracked: %s", path)
		}
	}
}

func TestVerifyUsesNativeGuardrails(t *testing.T) {
	root := repositoryRoot(t)
	makefile, err := os.ReadFile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(makefile), "verify-guardrails.sh") {
		t.Fatal("Makefile still invokes a shell guardrail")
	}
	if !strings.Contains(string(makefile), "go test ./test/architecture") {
		t.Fatal("Makefile verify target does not run native architecture guardrails")
	}
	if _, err := os.Stat(filepath.Join(root, "scripts", "verify-guardrails.sh")); !os.IsNotExist(err) {
		t.Fatal("obsolete shell guardrail still exists")
	}
}
