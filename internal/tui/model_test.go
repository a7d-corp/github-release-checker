package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/a7d-corp/github-release-checker/internal/config"
	"github.com/a7d-corp/github-release-checker/internal/github"
)

var (
	repoA = config.Repo{Owner: "acme", Name: "alpha"}
	repoB = config.Repo{Owner: "acme", Name: "beta"}
)

func noopCheck(_ context.Context, r config.Repo) github.Result {
	return github.Result{Repo: r}
}

func newTestModel() Model {
	return New([]config.Repo{repoA, repoB}, noopCheck)
}

func update(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func TestNewRowsAreLoading(t *testing.T) {
	m := newTestModel()

	rows := m.table.Rows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	for _, row := range rows {
		if row[1] != "loading…" {
			t.Errorf("row %v: want loading…", row)
		}
	}
	if !strings.Contains(m.View(), "Checking 0/2") {
		t.Errorf("view does not show progress:\n%s", m.View())
	}
}

func TestResultUpdatesCorrectRow(t *testing.T) {
	m := newTestModel()

	m, _ = update(t, m, resultMsg{gen: m.gen, result: github.Result{
		Repo:         repoB,
		Tag:          "v1.2.3",
		PublishedAt:  time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC),
		CommitsSince: 7,
		AutoRelease:  true,
	}})

	rows := m.table.Rows()
	want := []string{"acme/beta", "v1.2.3", "2026-09-01", "7", "automatic"}
	for i := range want {
		if rows[0][i] != want[i] {
			t.Errorf("row 0 col %d: got %q, want %q", i, rows[0][i], want[i])
		}
	}
	if rows[1][1] != "loading…" {
		t.Errorf("row 1: got %v, want loading…", rows[1])
	}
	if !strings.Contains(m.View(), "Checking 1/2") {
		t.Errorf("view does not show progress:\n%s", m.View())
	}
}

func TestRowsSortedByCommitsSince(t *testing.T) {
	repos := []config.Repo{
		{Owner: "acme", Name: "loading"},
		{Owner: "acme", Name: "error"},
		{Owner: "acme", Name: "none"},
		{Owner: "acme", Name: "few"},
		{Owner: "acme", Name: "many"},
		{Owner: "acme", Name: "few-too"},
	}
	m := New(repos, noopCheck)

	for _, res := range []github.Result{
		{Repo: repos[1], Err: errors.New("boom")},
		{Repo: repos[2], NoRelease: true},
		{Repo: repos[3], Tag: "v1", CommitsSince: 3},
		{Repo: repos[4], Tag: "v1", CommitsSince: 40},
		{Repo: repos[5], Tag: "v1", CommitsSince: 3},
	} {
		m, _ = update(t, m, resultMsg{gen: m.gen, result: res})
	}

	want := []string{"acme/many", "acme/few", "acme/few-too", "acme/none", "acme/error", "acme/loading"}
	rows := m.table.Rows()
	for i := range want {
		if rows[i][0] != want[i] {
			t.Errorf("row %d: got %q, want %q", i, rows[i][0], want[i])
		}
	}
}

func TestNoReleaseRow(t *testing.T) {
	m := newTestModel()

	m, _ = update(t, m, resultMsg{gen: m.gen, result: github.Result{Repo: repoA, NoRelease: true}})

	want := []string{"acme/alpha", "none", "none", "-", "manual"}
	for i := range want {
		if got := m.table.Rows()[0][i]; got != want[i] {
			t.Errorf("col %d: got %q, want %q", i, got, want[i])
		}
	}
}

func TestErrorRow(t *testing.T) {
	m := newTestModel()

	m, _ = update(t, m, resultMsg{gen: m.gen, result: github.Result{Repo: repoA, Err: errors.New("boom")}})

	if got := m.table.Rows()[0][1]; got != "error" {
		t.Errorf("tag col: got %q, want error", got)
	}
	if !strings.Contains(m.View(), "acme/alpha: boom") {
		t.Errorf("view does not show the error:\n%s", m.View())
	}
}

func TestAllDoneHidesProgress(t *testing.T) {
	m := newTestModel()

	m, _ = update(t, m, resultMsg{gen: m.gen, result: github.Result{Repo: repoA, NoRelease: true}})
	m, _ = update(t, m, resultMsg{gen: m.gen, result: github.Result{Repo: repoB, NoRelease: true}})

	if strings.Contains(m.View(), "Checking") {
		t.Errorf("view still shows progress:\n%s", m.View())
	}
}

func TestRefreshResetsAndIgnoresStaleResults(t *testing.T) {
	m := newTestModel()
	oldGen := m.gen
	m, _ = update(t, m, resultMsg{gen: oldGen, result: github.Result{Repo: repoA, NoRelease: true}})

	m, cmd := update(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Error("refresh returned no command")
	}
	if got := m.table.Rows()[0][1]; got != "loading…" {
		t.Errorf("after refresh: got %q, want loading…", got)
	}

	m, _ = update(t, m, resultMsg{gen: oldGen, result: github.Result{Repo: repoA, NoRelease: true}})
	if got := m.table.Rows()[0][1]; got != "loading…" {
		t.Errorf("stale result was applied: got %q", got)
	}
}

func TestQuitKeys(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyCtrlC},
	} {
		_, cmd := update(t, newTestModel(), key)
		if cmd == nil {
			t.Fatalf("%s: no command", key)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s: command does not quit", key)
		}
	}
}

func TestCheckCmd(t *testing.T) {
	m := New([]config.Repo{repoA}, func(_ context.Context, r config.Repo) github.Result {
		return github.Result{Repo: r, Tag: "v9"}
	})

	msg, ok := m.checkCmd(repoA)().(resultMsg)
	if !ok {
		t.Fatal("command did not return a resultMsg")
	}
	if msg.gen != m.gen || msg.result.Tag != "v9" {
		t.Errorf("got %+v", msg)
	}
}

func TestInitReturnsCommand(t *testing.T) {
	if newTestModel().Init() == nil {
		t.Error("Init returned no command")
	}
}

func TestWindowSizeSetsTableHeight(t *testing.T) {
	m := New([]config.Repo{repoA, repoB}, noopCheck)

	m, _ = update(t, m, tea.WindowSizeMsg{Width: 80, Height: 5})

	if m.table.Height() < 1 || m.table.Height() > 5 {
		t.Errorf("table height %d does not fit window", m.table.Height())
	}
}
