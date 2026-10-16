// Package trace implements Phase 1C — Trace Continuity Enforcement.
//
// A TraceContext carries trace_id immutably from the moment it is injected.
// Once set, trace_id CANNOT be changed — any drift is a HARD FAIL.
//
// Every layer that receives a TraceContext must pass the SAME instance forward.
// No layer may create a new TraceContext with a different trace_id.
//
// Trace break detection:
//   TraceContext.AssertContinuity(traceID, stage) verifies the ID has not drifted.
//   If it has drifted → structured TraceBreakError is returned with expected/got/stage.
package trace

import (
	"fmt"
	"log"
	"sync"
)

// TraceBreakError is returned when a trace_id mismatch is detected.
// This is a HARD FAIL — no transaction proceeds with a broken trace.
type TraceBreakError struct {
	Expected string
	Got      string
	Stage    string
}

func (e *TraceBreakError) Error() string {
	return fmt.Sprintf("[TRACE BREAK] stage=%s expected=%s got=%s", e.Stage, e.Expected, e.Got)
}

// Context carries an immutable trace_id across the execution pipeline.
// Once injected, the ID cannot be changed.
type Context struct {
	mu      sync.RWMutex
	traceID string
	locked  bool
}

// New creates a new TraceContext. If traceID is non-empty it is locked immediately.
func New(traceID string) *Context {
	c := &Context{}
	if traceID != "" {
		c.traceID = traceID
		c.locked = true
	}
	return c
}

// Inject sets the trace_id for the first and only time.
// If already set with a different value, returns TraceBreakError.
func (c *Context) Inject(traceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked {
		if c.traceID != traceID {
			err := &TraceBreakError{Expected: c.traceID, Got: traceID, Stage: "INJECT"}
			log.Printf("[TRACE][BREAK] %s", err.Error())
			return err
		}
		return nil
	}
	c.traceID = traceID
	c.locked = true
	log.Printf("[TRACE][INJECT] trace_id=%s", traceID)
	return nil
}

// ID returns the current trace_id.
func (c *Context) ID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.traceID
}

// AssertContinuity verifies the trace_id has not drifted at a given stage.
// Returns TraceBreakError if the ID does not match.
func (c *Context) AssertContinuity(traceID, stage string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.traceID != traceID {
		err := &TraceBreakError{Expected: c.traceID, Got: traceID, Stage: stage}
		log.Printf("[TRACE][BREAK] %s", err.Error())
		return err
	}
	log.Printf("[TRACE][CONTINUITY OK] stage=%s trace_id=%s", stage, traceID)
	return nil
}

// LogStage logs a structured trace event for a given stage.
// Every layer MUST call this so trace continuity is observable in logs.
func (c *Context) LogStage(stage, detail string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	log.Printf("[TRACE][%s] trace_id=%s %s", stage, c.traceID, detail)
}
