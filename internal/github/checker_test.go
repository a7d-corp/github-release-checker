package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/a7d-corp/github-release-checker/internal/config"
)

var repo = config.Repo{Owner: "acme", Name: "widget"}

func newTestChecker(t *testing.T, mux *http.ServeMux) *Checker {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c, err := NewChecker("test-token", srv.URL+"/")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func repoHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(`{"default_branch":"main"}`))
}

func TestCheckWithRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/acme/widget", repoHandler)
	mux.HandleFunc("GET /repos/acme/widget/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","published_at":"2026-09-01T10:00:00Z"}`))
	})
	mux.HandleFunc("GET /repos/acme/widget/compare/v1.2.3...main", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ahead_by":7}`))
	})

	got := newTestChecker(t, mux).Check(context.Background(), repo)

	if got.Err != nil {
		t.Fatalf("unexpected error: %v", got.Err)
	}
	if got.Repo != repo {
		t.Errorf("Repo: got %+v, want %+v", got.Repo, repo)
	}
	if got.NoRelease {
		t.Error("NoRelease: got true, want false")
	}
	if got.Tag != "v1.2.3" {
		t.Errorf("Tag: got %q, want %q", got.Tag, "v1.2.3")
	}
	if want := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC); !got.PublishedAt.Equal(want) {
		t.Errorf("PublishedAt: got %v, want %v", got.PublishedAt, want)
	}
	if got.CommitsSince != 7 {
		t.Errorf("CommitsSince: got %d, want 7", got.CommitsSince)
	}
	if got.AutoRelease {
		t.Error("AutoRelease: got true, want false")
	}
}

func TestCheckAutoRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/acme/widget", repoHandler)
	mux.HandleFunc("GET /repos/acme/widget/contents/.github/workflows/zz_generated.auto_release.yaml", func(w http.ResponseWriter, r *http.Request) {
		if ref := r.URL.Query().Get("ref"); ref != "main" {
			t.Errorf("ref: got %q, want main", ref)
		}
		_, _ = w.Write([]byte(`{"type":"file","name":"zz_generated.auto_release.yaml"}`))
	})
	mux.HandleFunc("GET /repos/acme/widget/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	got := newTestChecker(t, mux).Check(context.Background(), repo)

	if got.Err != nil {
		t.Fatalf("unexpected error: %v", got.Err)
	}
	if !got.AutoRelease {
		t.Error("AutoRelease: got false, want true")
	}
	if !got.NoRelease {
		t.Error("NoRelease: got false, want true")
	}
}

func TestCheckNoRelease(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /repos/acme/widget", repoHandler)
	mux.HandleFunc("GET /repos/acme/widget/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	got := newTestChecker(t, mux).Check(context.Background(), repo)

	if got.Err != nil {
		t.Fatalf("unexpected error: %v", got.Err)
	}
	if !got.NoRelease {
		t.Error("NoRelease: got false, want true")
	}
}

func TestCheckErrors(t *testing.T) {
	fail := func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	}
	release := func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v1.0.0","published_at":"2026-09-01T10:00:00Z"}`))
	}

	tests := map[string]func(*http.ServeMux){
		"repo fails": func(mux *http.ServeMux) {
			mux.HandleFunc("GET /repos/acme/widget", fail)
		},
		"workflow lookup fails": func(mux *http.ServeMux) {
			mux.HandleFunc("GET /repos/acme/widget", repoHandler)
			mux.HandleFunc("GET /repos/acme/widget/contents/.github/workflows/zz_generated.auto_release.yaml", fail)
		},
		"release fails": func(mux *http.ServeMux) {
			mux.HandleFunc("GET /repos/acme/widget", repoHandler)
			mux.HandleFunc("GET /repos/acme/widget/releases/latest", fail)
		},
		"compare fails": func(mux *http.ServeMux) {
			mux.HandleFunc("GET /repos/acme/widget", repoHandler)
			mux.HandleFunc("GET /repos/acme/widget/releases/latest", release)
			mux.HandleFunc("GET /repos/acme/widget/compare/v1.0.0...main", fail)
		},
	}
	for name, setup := range tests {
		t.Run(name, func(t *testing.T) {
			mux := http.NewServeMux()
			setup(mux)

			got := newTestChecker(t, mux).Check(context.Background(), repo)

			if got.Err == nil {
				t.Error("expected an error, got nil")
			}
			if got.Repo != repo {
				t.Errorf("Repo: got %+v, want %+v", got.Repo, repo)
			}
		})
	}
}
