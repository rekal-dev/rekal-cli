package nomic

import (
	"testing"
	"time"
)

// TestIdleVerdict_InFlightRequestBlocksShutdown is the regression this exists
// for.
//
// The idle timer was reset when a request was *read*, not when it finished.
// Embedding a batch of 8192-token documents on CPU runs for minutes, so one
// request outlived the 5-minute timeout and the daemon returned from its loop
// mid-request — running the deferred embedder.Close() and ln.Close() while a
// handler goroutine was still inside the model. Observed as 353 "broken pipe"
// warnings in a single embed run, each one a discarded batch, and freeing a
// llama context under a live inference is the likeliest source of the one
// segfault seen.
func TestIdleVerdict_InFlightRequestBlocksShutdown(t *testing.T) {
	t.Parallel()

	now := time.Now()
	// A request read an hour ago and still running: far past any timeout.
	shutdown, wait := idleVerdict(1, now.Add(-time.Hour), now, 5*time.Minute)
	if shutdown {
		t.Error("daemon would shut down with a request in flight — that tears the model down under a live handler")
	}
	if wait <= 0 {
		t.Errorf("wait is %v, want a positive re-arm so the timer fires again", wait)
	}
}

// TestIdleVerdict_ClockRunsFromCompletion pins the second rule: a long request
// that has just finished must not be treated as idle time already served.
func TestIdleVerdict_ClockRunsFromCompletion(t *testing.T) {
	t.Parallel()

	now := time.Now()
	const timeout = 5 * time.Minute

	// Nothing in flight, but the last request completed one second ago.
	shutdown, wait := idleVerdict(0, now.Add(-time.Second), now, timeout)
	if shutdown {
		t.Error("daemon would shut down one second after finishing a request")
	}
	if wait > timeout || wait <= 0 {
		t.Errorf("wait is %v, want the remainder of the timeout (0 < w <= %v)", wait, timeout)
	}
}

// TestIdleVerdict_TrulyIdleShutsDown keeps the feature honest: the daemon still
// exits when it is genuinely unused, which is why it is disposable at all.
func TestIdleVerdict_TrulyIdleShutsDown(t *testing.T) {
	t.Parallel()

	now := time.Now()
	const timeout = 5 * time.Minute

	shutdown, _ := idleVerdict(0, now.Add(-timeout-time.Second), now, timeout)
	if !shutdown {
		t.Error("an idle daemon with no work must still shut down; otherwise it leaks a process and the model's memory per store")
	}

	// Exactly at the boundary counts as expired.
	if s, _ := idleVerdict(0, now.Add(-timeout), now, timeout); !s {
		t.Error("at exactly the timeout the daemon should shut down")
	}
}
