package kitchen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashRollIsDeterministicAndBounded(t *testing.T) {
	t.Parallel()
	for _, seed := range []string{"a", "pancakes-1", "kitchen|bake|brunch|2"} {
		first := HashRoll(seed)
		second := HashRoll(seed)
		if first != second {
			t.Errorf("HashRoll(%q) not deterministic: %f != %f", seed, first, second)
		}
		if first < 0 || first >= 1 {
			t.Errorf("HashRoll(%q) = %f, outside [0,1)", seed, first)
		}
	}
	if HashRoll("seed-a") == HashRoll("seed-b") {
		t.Error("HashRoll returned identical values for different seeds")
	}
}

func TestChaosEngineNeverFailsAtLevelZero(t *testing.T) {
	t.Parallel()
	engine := NewChaosEngine(0.0, "home-kitchen", NewMemoryStore())
	for _, rt := range []string{"oven_bake", "oven_ingredient"} {
		for i := range 20 {
			if err := engine.Roll(rt, fmt.Sprintf("r%d", i)); err != nil {
				t.Fatalf("Roll at level 0 failed: %v", err)
			}
		}
	}
}

func TestChaosEngineAlwaysFailsAtLevelOneBeforeCap(t *testing.T) {
	t.Parallel()
	engine := NewChaosEngine(1.0, "home-kitchen", NewMemoryStore())
	var lastErr *FailureError
	for attempt := 1; attempt < MaxAttempts; attempt++ {
		err := engine.Roll("oven_bake", "sourdough")
		if err == nil {
			t.Fatalf("attempt %d: expected failure at level 1.0", attempt)
		}
		var fe *FailureError
		if !errors.As(err, &fe) {
			t.Fatalf("expected *FailureError, got %T", err)
		}
		if fe.Attempt != attempt {
			t.Errorf("attempt %d: FailureError.Attempt = %d", attempt, fe.Attempt)
		}
		if fe.Message == "" {
			t.Error("FailureError.Message is empty")
		}
		lastErr = fe
	}
	if lastErr == nil || !strings.Contains(lastErr.Error(), fmt.Sprintf("attempt %d", MaxAttempts-1)) {
		t.Errorf("error text should mention attempt number, got: %v", lastErr)
	}
}

func TestChaosEngineAlwaysSucceedsAtCap(t *testing.T) {
	t.Parallel()
	engine := NewChaosEngine(1.0, "home-kitchen", NewMemoryStore())
	// Drive right up to the cap: two guaranteed failures.
	_ = engine.Roll("oven_bake", "sourdough")
	_ = engine.Roll("oven_bake", "sourdough")
	if err := engine.Roll("oven_bake", "sourdough"); err != nil {
		t.Fatalf("roll at attempt %d should always succeed, got: %v", MaxAttempts, err)
	}
}

func TestChaosEngineStreakResetsOnSuccessAndPersists(t *testing.T) {
	t.Parallel()
	store := NewMemoryStore()

	// A successful attempt resets the streak to zero.
	successful := NewChaosEngine(0.5, "home-kitchen", store)
	successful.roll = func(string) float64 { return 0.9 }
	if err := successful.Roll("oven_bake", "cake"); err != nil {
		t.Fatalf("roll above level should succeed: %v", err)
	}
	if n, ok := store.Load("home-kitchen|oven_bake|cake"); ok && n != 0 {
		t.Errorf("streak after success = %d (present: %v); want reset to 0", n, ok)
	}

	// A failure at attempt 1 persists for the next engine to see.
	failing := NewChaosEngine(0.5, "home-kitchen", store)
	failing.roll = func(string) float64 { return 0.1 }
	if err := failing.Roll("oven_bake", "cake"); err == nil {
		t.Fatal("expected failure on first attempt")
	}
	if n, _ := store.Load("home-kitchen|oven_bake|cake"); n != 1 {
		t.Errorf("streak after failure = %d; want 1", n)
	}

	// A later engine continues the streak and lands on the cap.
	second := NewChaosEngine(0.5, "home-kitchen", store)
	second.roll = func(string) float64 { return 0.1 }
	if err := second.Roll("oven_bake", "cake"); err == nil {
		t.Fatal("expected failure on attempt 2")
	}
	if err := second.Roll("oven_bake", "cake"); err != nil {
		t.Fatalf("attempt %d must always succeed: %v", MaxAttempts, err)
	}
	if n, _ := store.Load("home-kitchen|oven_bake|cake"); n != 0 {
		t.Errorf("streak after cap success = %d; want 0", n)
	}
}

func TestMemoryStoreLoadSetRoundTrip(t *testing.T) {
	t.Parallel()
	s := NewMemoryStore()
	if n, ok := s.Load("missing"); ok || n != 0 {
		t.Errorf("missing key Load = %d, %v; want 0, false", n, ok)
	}
	s.Set("k", 7)
	if n, ok := s.Load("k"); !ok || n != 7 {
		t.Errorf("Load = %d, %v; want 7, true", n, ok)
	}
}

func TestFileStorePersistsAcrossInstances(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "kitchen.json")

	first := NewFileStore(path)
	if err := first.Set("home-kitchen|oven_bake|bread", 2); err != nil {
		t.Fatalf("Set: %v", err)
	}

	second := NewFileStore(path)
	if n, ok := second.Load("home-kitchen|oven_bake|bread"); !ok || n != 2 {
		t.Errorf("reopened FileStore Load = %d, %v; want 2, true", n, ok)
	}
	if n, ok := second.Load("other"); ok || n != 0 {
		t.Errorf("unknown key Load = %d, %v; want 0, false", n, ok)
	}
}

func TestFileStoreCreatesParentDirectories(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "deep", "kitchen.json")
	s := NewFileStore(path)
	if err := s.Set("k", 1); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if n, ok := NewFileStore(path).Load("k"); !ok || n != 1 {
		t.Errorf("Load after nested Set = %d, %v; want 1, true", n, ok)
	}
}

func TestFileStoreRoundTripCorruptFile(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "broken.json")
	if err := os.WriteFile(path, []byte("this is not json"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	s := NewFileStore(path)
	if n, ok := s.Load("anything"); ok || n != 0 {
		t.Errorf("corrupt file should behave like empty state, got %d, %v", n, ok)
	}
	if err := s.Set("k", 3); err != nil {
		t.Fatalf("Set should repair corrupt file: %v", err)
	}
	if n, ok := s.Load("k"); !ok || n != 3 {
		t.Errorf("Load after repair = %d, %v; want 3, true", n, ok)
	}
}

func TestMessageForIsDeterministicAndCatalogued(t *testing.T) {
	t.Parallel()
	if len(FailureMessages) < 10 {
		t.Errorf("want a rich failure catalog, got %d messages", len(FailureMessages))
	}
	seen := map[string]bool{}
	for i := range 50 {
		msg := MessageFor(fmt.Sprintf("seed-%d", i))
		if msg == "" {
			t.Fatalf("MessageFor produced empty message")
		}
		if !slicesContains(FailureMessages, msg) {
			t.Errorf("MessageFor(%q) = %q, not in catalog", fmt.Sprintf("seed-%d", i), msg)
		}
		seen[msg] = true
	}
	if len(seen) < 3 {
		t.Errorf("MessageFor should spread across catalog, saw only %d", len(seen))
	}
	if MessageFor("same-seed") != MessageFor("same-seed") {
		t.Error("MessageFor not deterministic")
	}
}

func slicesContains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}