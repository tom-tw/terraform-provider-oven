// Package kitchen holds the pure domain logic of the oven provider:
// the chaos engine that simulates quirky kitchen failures, unit and
// temperature conversions, and the provider's recipe book.
package kitchen

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// MaxAttempts is the first attempt that is guaranteed to succeed: no
// matter what the chaos engine rolls, the kitchen lets you finish on
// the third try so nobody gets stuck in an infinite failure loop.
const MaxAttempts = 3

// FailureError is returned when the chaos engine decides an operation
// fails. It carries the themed Message and the human-visible Attempt.
type FailureError struct {
	Resource string
	Attempt  int
	Message  string
}

func (e *FailureError) Error() string {
	return fmt.Sprintf("%s: %s (kitchen attempt %d: another try may succeed)", e.Resource, e.Message, e.Attempt)
}

// Store persists per-resource attempt counters so retries behave
// consistently across separate OpenTofu runs. Set errors are handled
// best-effort by the engine: a lost counter only resets the retry
// streak, it never corrupts OpenTofu state.
type Store interface {
	Load(key string) (int, bool)
	Set(key string, value int) error
}

// MemoryStore keeps counters in RAM. Counters vanish between runs,
// which is only useful for tests.
type MemoryStore struct {
	counters map[string]int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{counters: map[string]int{}}
}

func (s *MemoryStore) Load(key string) (int, bool) {
	n, ok := s.counters[key]
	return n, ok
}

func (s *MemoryStore) Set(key string, value int) error {
	s.counters[key] = value
	return nil
}

// ChaosEngine decides whether a kitchen operation randomly fails.
// Rolls are deterministic per (kitchen, resource, attempt), so the
// same apply reproduces the same outcome, while a retry moves to a
// fresh attempt and usually succeeds.
type ChaosEngine struct {
	Level   float64 // probability of a random failure, 0..1
	Kitchen string
	Store   Store

	roll func(seed string) float64
}

func NewChaosEngine(level float64, kitchenName string, store Store) *ChaosEngine {
	if level < 0 {
		level = 0
	}
	if level > 1 {
		level = 1
	}
	return &ChaosEngine{
		Level:   level,
		Kitchen: kitchenName,
		Store:   store,
		roll:    HashRoll,
	}
}

func (e *ChaosEngine) key(resourceType, resourceName string, attempt int) string {
	return fmt.Sprintf("%s|%s|%s|%d", e.Kitchen, resourceType, resourceName, attempt)
}

// nextAttempt increments and persists the failure-streak counter for
// this resource and returns the new attempt number (1-based).
func (e *ChaosEngine) nextAttempt(resourceType, resourceName string) int {
	if e.Store == nil {
		return 1 // without persistence every roll is a first attempt
	}
	key := fmt.Sprintf("%s|%s|%s", e.Kitchen, resourceType, resourceName)
	n, _ := e.Store.Load(key)
	n++
	_ = e.Store.Set(key, n) // best-effort: losing a counter only resets the streak
	return n
}

// reset clears the failure-streak counter once a kitchen operation
// finally succeeds, so future chaos starts fresh at attempt 1.
func (e *ChaosEngine) reset(resourceType, resourceName string) {
	if e.Store == nil {
		return
	}
	key := fmt.Sprintf("%s|%s|%s", e.Kitchen, resourceType, resourceName)
	_ = e.Store.Set(key, 0) // best-effort
}

// Roll decides whether the operation on this resource fails right
// now. The counter tracks consecutive unsuccessful attempts: attempt
// 3 always succeeds and resets the streak; attempts below MaxAttempts
// fail with probability Level, with a themed message.
func (e *ChaosEngine) Roll(resourceType, resourceName string) error {
	if e.Level <= 0 {
		return nil
	}
	resource := resourceType + "." + resourceName
	if e.Kitchen != "" {
		resource = e.Kitchen + "/" + resource
	}

	attempt := e.nextAttempt(resourceType, resourceName)
	if attempt >= MaxAttempts {
		e.reset(resourceType, resourceName)
		return nil
	}

	if e.roll(e.key(resourceType, resourceName, attempt)) >= e.Level {
		e.reset(resourceType, resourceName)
		return nil
	}

	return &FailureError{
		Resource: resource,
		Attempt:  attempt,
		Message:  MessageFor(e.key(resourceType, resourceName, attempt)),
	}
}

// HashRoll maps a seed string onto [0,1) deterministically.
func HashRoll(seed string) float64 {
	sum := sha256.Sum256([]byte(seed))
	u := binary.BigEndian.Uint64(sum[:8])
	return float64(u) / float64(1<<64)
}