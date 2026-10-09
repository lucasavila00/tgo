package navigation

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestConcurrentColdRequestsShareOneBuild(t *testing.T) {
	engine, err := New(filepath.Join("testdata", "workspaces", "cross-package"))
	if err != nil {
		t.Fatal(err)
	}
	errors := make(chan error, 8)
	for range 8 {
		go func() {
			_, err := engine.WorkspaceSymbols(context.Background(), "")
			errors <- err
		}()
	}
	for range 8 {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.index == nil || engine.building || engine.ready != nil {
		t.Fatalf(
			"cold build state = index %v, building %t, ready %v",
			engine.index != nil, engine.building, engine.ready,
		)
	}
}

func TestInvalidationRejectsInFlightGeneration(t *testing.T) {
	engine, err := New(filepath.Join("testdata", "workspaces", "cross-package"))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := engine.WorkspaceSymbols(context.Background(), "")
		done <- err
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		engine.mu.Lock()
		building := engine.building
		engine.mu.Unlock()
		if building {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("workspace build did not start")
		}
		time.Sleep(time.Millisecond)
	}
	engine.Invalidate()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	if engine.generation != 1 || engine.index == nil || engine.building {
		t.Fatalf(
			"invalidated build state = generation %d, index %v, building %t",
			engine.generation, engine.index != nil, engine.building,
		)
	}
}
