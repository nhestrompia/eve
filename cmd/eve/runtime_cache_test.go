package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRuntimeDerivedCacheSharesConcurrentLoads(t *testing.T) {
	cache := newRuntimeDerivedCache()
	var loads atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	load := func() (string, error) {
		if loads.Add(1) == 1 {
			close(started)
		}
		<-release
		return "derived", nil
	}

	const callers = 32
	var wait sync.WaitGroup
	wait.Add(callers)
	results := make(chan string, callers)
	for range callers {
		go func() {
			defer wait.Done()
			value, err := cachedRuntimeValue(
				context.Background(),
				cache,
				"repository:fixture",
				1,
				time.Minute,
				load,
			)
			if err != nil {
				t.Errorf("cachedRuntimeValue: %v", err)
				return
			}
			results <- value
		}()
	}
	<-started
	time.Sleep(20 * time.Millisecond)
	close(release)
	wait.Wait()
	close(results)

	for value := range results {
		if value != "derived" {
			t.Fatalf("value = %q, want derived", value)
		}
	}
	if got := loads.Load(); got != 1 {
		t.Fatalf("loads = %d, want 1", got)
	}
}

func TestRuntimeHealthReportsFanoutDiagnostics(t *testing.T) {
	server := newRuntimeServer(repository{ID: "fixture", Root: t.TempDir()}, "localhost:0")
	stream, unsubscribe := server.events.subscribe()
	defer unsubscribe()
	_ = stream
	if _, err := cachedRuntimeValue(
		context.Background(),
		server.derivedCache,
		"fixture",
		server.events.currentGeneration(),
		time.Minute,
		func() (string, error) { return "cached", nil },
	); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	body := recorder.Body.String()
	for _, expected := range []string{
		`"eventSubscribers":1`,
		`"watchedDirectories":0`,
		`"entries":1`,
		`"inFlight":0`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("health response %s does not contain %s", body, expected)
		}
	}
}

func TestRuntimeDerivedCacheInvalidatesOnGenerationChange(t *testing.T) {
	cache := newRuntimeDerivedCache()
	var loads atomic.Int32
	load := func() (int32, error) {
		return loads.Add(1), nil
	}

	first, err := cachedRuntimeValue(context.Background(), cache, "repos", 1, time.Minute, load)
	if err != nil {
		t.Fatal(err)
	}
	reused, err := cachedRuntimeValue(context.Background(), cache, "repos", 1, time.Minute, load)
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := cachedRuntimeValue(context.Background(), cache, "repos", 2, time.Minute, load)
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || reused != 1 || refreshed != 2 {
		t.Fatalf("values = %d, %d, %d; want 1, 1, 2", first, reused, refreshed)
	}
}
