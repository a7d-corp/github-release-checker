// Command github-release-checker shows the latest GitHub release of each
// configured repository and the number of commits since that release.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/a7d-corp/github-release-checker/internal/config"
	"github.com/a7d-corp/github-release-checker/internal/github"
	"github.com/a7d-corp/github-release-checker/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	defaultPath, err := config.DefaultPath()
	if err != nil {
		return err
	}
	configPath := flag.String("config", defaultPath, "path to the repositories config file")
	flag.Parse()

	repos, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	if token == "" {
		return errors.New("set GITHUB_TOKEN or GH_TOKEN to a GitHub token")
	}

	checker, err := github.NewChecker(token, "")
	if err != nil {
		return err
	}

	_, err = tea.NewProgram(tui.New(repos, checker.Check), tea.WithAltScreen()).Run()
	return err
}
