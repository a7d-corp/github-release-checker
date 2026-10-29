// Package config loads the list of repositories to check.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Repo identifies a GitHub repository.
type Repo struct {
	Owner string
	Name  string
}

// String returns the repository in owner/name form.
func (r Repo) String() string {
	return r.Owner + "/" + r.Name
}

type file struct {
	Repos []string `yaml:"repos"`
}

// DefaultPath returns $HOME/.config/github-release-checker/repositories.yaml.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "github-release-checker", "repositories.yaml"), nil
}

// Load reads the config file at path and returns its repositories in file order.
func Load(path string) ([]Repo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var f file
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if len(f.Repos) == 0 {
		return nil, errors.New("config " + path + " has no repos")
	}

	repos := make([]Repo, 0, len(f.Repos))
	for _, entry := range f.Repos {
		owner, name, ok := strings.Cut(entry, "/")
		if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
			return nil, fmt.Errorf("config %s: invalid repo %q, want owner/name", path, entry)
		}
		repos = append(repos, Repo{Owner: owner, Name: name})
	}
	return repos, nil
}
