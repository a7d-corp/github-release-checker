// Package tui shows the release status of repositories in a table.
package tui

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/a7d-corp/github-release-checker/internal/config"
	"github.com/a7d-corp/github-release-checker/internal/github"
)

const (
	dateFormat = "2006-01-02"
	// chromeHeight is the number of lines the view uses outside the table:
	// the title, a blank line, the help line and one spare line.
	chromeHeight = 4
	// headerHeight is the number of lines the table header uses.
	headerHeight = 1
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// CheckFunc returns the release status of one repository.
type CheckFunc func(context.Context, config.Repo) github.Result

// resultMsg carries a check result. gen identifies the refresh that started
// the check, so the model can drop results from an earlier refresh.
type resultMsg struct {
	gen    int
	result github.Result
}

// Model is the Bubble Tea model for the release table.
type Model struct {
	repos   []config.Repo
	check   CheckFunc
	results map[config.Repo]github.Result
	gen     int
	table   table.Model
	spinner spinner.Model
}

// New returns a Model for repos. check runs once for each repository.
func New(repos []config.Repo, check CheckFunc) Model {
	repoWidth := len("Repository")
	for _, r := range repos {
		repoWidth = max(repoWidth, len(r.String()))
	}

	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Repository", Width: repoWidth},
			{Title: "Release tag", Width: 16},
			{Title: "Release date", Width: 12},
			{Title: "Commits since", Width: 13},
		}),
		table.WithFocused(true),
		table.WithHeight(len(repos)+headerHeight),
	)

	m := Model{
		repos:   repos,
		check:   check,
		results: map[config.Repo]github.Result{},
		table:   t,
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
	}
	m.table.SetRows(m.rows())
	return m
}

// Init starts the spinner and one check for each repository.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.checkAll())
}

// Update handles key presses, window resizes, spinner ticks and check results.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.gen++
			m.results = map[config.Repo]github.Result{}
			m.table.SetRows(m.rows())
			return m, m.checkAll()
		}
	case tea.WindowSizeMsg:
		m.table.SetHeight(max(headerHeight+1, min(len(m.repos)+headerHeight, msg.Height-chromeHeight)))
		return m, nil
	case resultMsg:
		if msg.gen != m.gen {
			return m, nil
		}
		results := maps.Clone(m.results)
		results[msg.result.Repo] = msg.result
		m.results = results
		m.table.SetRows(m.rows())
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// View renders the table, the progress line, any errors and the key help.
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("GitHub release checker"))
	if done := len(m.results); done < len(m.repos) {
		fmt.Fprintf(&b, "  %s Checking %d/%d…", m.spinner.View(), done, len(m.repos))
	}
	b.WriteString("\n\n")
	b.WriteString(m.table.View())
	b.WriteString("\n")

	for _, r := range m.repos {
		if res, ok := m.results[r]; ok && res.Err != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf("%s: %v", r, res.Err)))
			b.WriteString("\n")
		}
	}

	b.WriteString(helpStyle.Render("↑/↓ scroll • r refresh • q quit"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) checkAll() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.repos))
	for _, r := range m.repos {
		cmds = append(cmds, m.checkCmd(r))
	}
	return tea.Batch(cmds...)
}

func (m Model) checkCmd(r config.Repo) tea.Cmd {
	gen, check := m.gen, m.check
	return func() tea.Msg {
		return resultMsg{gen: gen, result: check(context.Background(), r)}
	}
}

// sortedRepos returns the repositories with the most commits since their
// release first. Repositories without a commit count go last, in the order
// no release, error, loading. Ties keep config order.
func (m Model) sortedRepos() []config.Repo {
	rank := func(r config.Repo) (int, int) {
		res, ok := m.results[r]
		switch {
		case !ok:
			return 3, 0
		case res.Err != nil:
			return 2, 0
		case res.NoRelease:
			return 1, 0
		default:
			return 0, res.CommitsSince
		}
	}

	repos := slices.Clone(m.repos)
	slices.SortStableFunc(repos, func(a, b config.Repo) int {
		aGroup, aCommits := rank(a)
		bGroup, bCommits := rank(b)
		return cmp.Or(cmp.Compare(aGroup, bGroup), cmp.Compare(bCommits, aCommits))
	})
	return repos
}

// rows returns one table row for each repository, in sortedRepos order.
func (m Model) rows() []table.Row {
	rows := make([]table.Row, 0, len(m.repos))
	for _, r := range m.sortedRepos() {
		res, ok := m.results[r]
		switch {
		case !ok:
			rows = append(rows, table.Row{r.String(), "loading…", "", ""})
		case res.Err != nil:
			rows = append(rows, table.Row{r.String(), "error", "", ""})
		case res.NoRelease:
			rows = append(rows, table.Row{r.String(), "none", "none", "-"})
		default:
			rows = append(rows, table.Row{
				r.String(),
				res.Tag,
				res.PublishedAt.Format(dateFormat),
				strconv.Itoa(res.CommitsSince),
			})
		}
	}
	return rows
}
