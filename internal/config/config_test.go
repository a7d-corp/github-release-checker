package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "repositories.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValid(t *testing.T) {
	path := writeFile(t, `---
repos:
  - giantswarm/cloud-provider-cloud-director-app
  - giantswarm/cloud-provider-proxmox-app
`)

	repos, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []Repo{
		{Owner: "giantswarm", Name: "cloud-provider-cloud-director-app"},
		{Owner: "giantswarm", Name: "cloud-provider-proxmox-app"},
	}
	if len(repos) != len(want) {
		t.Fatalf("got %d repos, want %d", len(repos), len(want))
	}
	for i := range want {
		if repos[i] != want[i] {
			t.Errorf("repo %d: got %+v, want %+v", i, repos[i], want[i])
		}
	}
}

func TestLoadErrors(t *testing.T) {
	tests := map[string]string{
		"empty list":   "repos: []\n",
		"no repos key": "foo: bar\n",
		"no slash":     "repos:\n  - giantswarm\n",
		"empty owner":  "repos:\n  - /name\n",
		"empty name":   "repos:\n  - owner/\n",
		"extra part":   "repos:\n  - a/b/c\n",
		"bad yaml":     "repos: [\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(writeFile(t, content)); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Error("expected an error, got nil")
	}
}

func TestDefaultPath(t *testing.T) {
	t.Setenv("HOME", "/home/test")

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/home/test/.config/github-release-checker/repositories.yaml"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRepoString(t *testing.T) {
	r := Repo{Owner: "a", Name: "b"}
	if r.String() != "a/b" {
		t.Errorf("got %q, want %q", r.String(), "a/b")
	}
}
