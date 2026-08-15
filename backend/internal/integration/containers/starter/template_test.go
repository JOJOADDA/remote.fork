package starter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSeedCreatesRTLReactStarter(t *testing.T) {
	root := t.TempDir()
	if err := Seed(root, "أخبار الذكاء الاصطناعي"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"package.json", "vite.config.ts", "src/main.tsx", "src/App.tsx", "src/index.css", "AGENTS.md"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatalf("starter file %s: %v", name, err)
		}
	}
	index, err := os.ReadFile(filepath.Join(root, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), `lang="ar" dir="rtl"`) {
		t.Fatalf("index.html is missing RTL metadata: %s", index)
	}
}

func TestSeedDoesNotOverwriteExistingProject(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "package.json")
	if err := os.WriteFile(path, []byte(`{"name":"existing"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Seed(root, "New Project"); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != `{"name":"existing"}` {
		t.Fatalf("existing package.json was overwritten: %s", contents)
	}
}
