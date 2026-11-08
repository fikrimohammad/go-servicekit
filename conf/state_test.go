package conf

import "testing"

func TestStateString(t *testing.T) {
	tests := map[State]string{
		StateLive:     "live",
		StateDegraded: "degraded",
		StateClosed:   "closed",
	}
	for state, want := range tests {
		if got := state.String(); got != want {
			t.Errorf("%v.String() = %q, want %q", state, got, want)
		}
	}
}

func TestStateTransitions(t *testing.T) {
	core := &loaderCore{degraded: make(map[string]struct{}), state: StateLive}

	if got := core.State(); got != StateLive {
		t.Fatalf("initial State() = %q, want %q", got, StateLive)
	}

	core.markDegraded("a")
	if got := core.State(); got != StateDegraded {
		t.Errorf("after markDegraded State() = %q, want %q", got, StateDegraded)
	}

	core.clearDegraded("a")
	if got := core.State(); got != StateLive {
		t.Errorf("after clearDegraded State() = %q, want %q", got, StateLive)
	}

	core.setProviderDegraded(true)
	if got := core.State(); got != StateDegraded {
		t.Errorf("after provider degraded State() = %q, want %q", got, StateDegraded)
	}

	core.markDegraded("a")
	core.setProviderDegraded(false)
	if got := core.State(); got != StateDegraded {
		t.Errorf("with one key still degraded State() = %q, want %q", got, StateDegraded)
	}

	core.clearDegraded("a")
	if got := core.State(); got != StateLive {
		t.Errorf("after full recovery State() = %q, want %q", got, StateLive)
	}

	core.closed.Store(true)
	core.setState(StateClosed)
	core.markDegraded("a")
	if got := core.State(); got != StateClosed {
		t.Errorf("after close State() = %q, want %q", got, StateClosed)
	}
}
