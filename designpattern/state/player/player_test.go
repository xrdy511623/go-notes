package player_test

import (
	"testing"

	"go-notes/designpattern/state/player"
	"go-notes/designpattern/state/states"
)

func newPlayer() *player.Player {
	return player.New("test-track.mp3", states.IdleState{})
}

func TestNewPlayer_StartsIdle(t *testing.T) {
	p := newPlayer()
	if got := p.State().String(); got != "idle" {
		t.Fatalf("new player state = %q, want %q", got, "idle")
	}
	if got := p.Track(); got != "test-track.mp3" {
		t.Fatalf("track = %q, want %q", got, "test-track.mp3")
	}
}

func TestPlay_FromIdle_TransitionsToPlaying(t *testing.T) {
	p := newPlayer()
	p.Play()
	if got := p.State().String(); got != "playing" {
		t.Fatalf("state after Play = %q, want %q", got, "playing")
	}
}

func TestPause_FromPlaying_TransitionsToPaused(t *testing.T) {
	p := newPlayer()
	p.Play()
	p.Pause()
	if got := p.State().String(); got != "paused" {
		t.Fatalf("state after Pause = %q, want %q", got, "paused")
	}
}

func TestPlay_FromPaused_ResumesPlaying(t *testing.T) {
	p := newPlayer()
	p.Play()
	p.Pause()
	p.Play()
	if got := p.State().String(); got != "playing" {
		t.Fatalf("state after resume = %q, want %q", got, "playing")
	}
}

func TestStop_FromPlaying_TransitionsToIdle(t *testing.T) {
	p := newPlayer()
	p.Play()
	p.Stop()
	if got := p.State().String(); got != "idle" {
		t.Fatalf("state after Stop = %q, want %q", got, "idle")
	}
}

func TestStop_FromPaused_TransitionsToIdle(t *testing.T) {
	p := newPlayer()
	p.Play()
	p.Pause()
	p.Stop()
	if got := p.State().String(); got != "idle" {
		t.Fatalf("state after Stop from paused = %q, want %q", got, "idle")
	}
}

func TestPlay_WhenAlreadyPlaying_NoOp(t *testing.T) {
	p := newPlayer()
	p.Play()
	histBefore := len(p.History())
	p.Play()
	histAfter := len(p.History())
	if histAfter != histBefore {
		t.Fatalf("Play when playing should be no-op, history grew from %d to %d", histBefore, histAfter)
	}
	if got := p.State().String(); got != "playing" {
		t.Fatalf("state = %q, want %q", got, "playing")
	}
}

func TestPause_WhenIdle_NoOp(t *testing.T) {
	p := newPlayer()
	histBefore := len(p.History())
	p.Pause()
	histAfter := len(p.History())
	if histAfter != histBefore {
		t.Fatalf("Pause when idle should be no-op, history grew from %d to %d", histBefore, histAfter)
	}
	if got := p.State().String(); got != "idle" {
		t.Fatalf("state = %q, want %q", got, "idle")
	}
}

func TestStop_WhenIdle_NoOp(t *testing.T) {
	p := newPlayer()
	histBefore := len(p.History())
	p.Stop()
	histAfter := len(p.History())
	if histAfter != histBefore {
		t.Fatalf("Stop when idle should be no-op, history grew from %d to %d", histBefore, histAfter)
	}
	if got := p.State().String(); got != "idle" {
		t.Fatalf("state = %q, want %q", got, "idle")
	}
}

func TestHistory_RecordsTransitions(t *testing.T) {
	p := newPlayer()
	p.Play()  // idle -> playing
	p.Pause() // playing -> paused
	p.Play()  // paused -> playing
	p.Stop()  // playing -> idle

	expected := []string{"idle", "playing", "paused", "playing", "idle"}
	history := p.History()

	if len(history) != len(expected) {
		t.Fatalf("history length = %d, want %d; got %v", len(history), len(expected), history)
	}
	for i, want := range expected {
		if history[i] != want {
			t.Errorf("history[%d] = %q, want %q", i, history[i], want)
		}
	}
}

func TestHistory_ReturnsCopy(t *testing.T) {
	p := newPlayer()
	h := p.History()
	h[0] = "tampered"
	if p.History()[0] == "tampered" {
		t.Fatal("History() should return a defensive copy")
	}
}
