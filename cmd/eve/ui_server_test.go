package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/nhestrompia/eve"
)

func TestRecordedRepositoryArtifactRoute(t *testing.T) {
	root := initTempGitRepo(t)
	mustRunInRepo(t, root, []string{"init", "--no-agent-instructions"})

	artifactPath := filepath.Join(root, "output", "playwright", "page.png")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte("recorded image")
	if err := os.WriteFile(artifactPath, want, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.txt"), []byte("private"), 0o644); err != nil {
		t.Fatal(err)
	}
	absoluteLog := filepath.Join(t.TempDir(), "sample.log")
	if err := os.WriteFile(absoluteLog, []byte("absolute log"), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot := sampleSnapshot(
		"snap_artifact",
		"Artifact route",
		gitOutputForTest(t, root, "rev-parse", "HEAD"),
	)
	snapshot.Artifacts = []eve.Artifact{{
		Type:     "screenshot",
		Path:     "output/playwright/page.png",
		MimeType: "image/png",
	}, {
		Type: "log",
		Path: absoluteLog,
	}}
	writeSnapshot(t, root, snapshot)

	handler := newRuntimeServer(repoFromRoot(root), "localhost:0").routes()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(
			http.MethodGet,
			"/api/repos/"+filepath.Base(root)+"/files/output/playwright/page.png",
			nil,
		),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Body.Bytes(); string(got) != string(want) {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := recorder.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q, want image/png", got)
	}

	assertRequestStatus(
		t,
		handler,
		http.MethodGet,
		"/api/repos/"+filepath.Base(root)+"/files/secret.txt",
		http.StatusNotFound,
		"artifact not found",
	)
	assertRequestStatus(
		t,
		handler,
		http.MethodGet,
		"/api/repos/"+filepath.Base(root)+"/files?path="+url.QueryEscape(filepath.Join(t.TempDir(), "unrecorded.log")),
		http.StatusNotFound,
		"artifact not found",
	)

	recorder = httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/repos/"+filepath.Base(root)+"/files?path="+url.QueryEscape(absoluteLog),
		nil,
	)
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "absolute log" {
		t.Fatalf("absolute log status = %d body = %q", recorder.Code, recorder.Body.String())
	}
}
