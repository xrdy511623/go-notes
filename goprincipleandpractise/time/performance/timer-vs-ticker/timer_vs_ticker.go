package timervsticker

import "time"

// SimulateTimerLoop creates a new Timer for each iteration, like using
// time.After in a select loop. This measures the allocation overhead of
// repeatedly creating timers.
func SimulateTimerLoop(iterations int) {
	for i := 0; i < iterations; i++ {
		timer := time.NewTimer(time.Hour) // 模拟 time.After
		timer.Stop()
	}
}

// SimulateTickerReuse creates a single Ticker and reuses it across iterations.
// This demonstrates the benefit of timer reuse.
func SimulateTickerReuse(iterations int) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for i := 0; i < iterations; i++ {
		ticker.Reset(time.Hour)
	}
}

// SimulateTimerReuse creates a single Timer and resets it across iterations.
func SimulateTimerReuse(iterations int) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for i := 0; i < iterations; i++ {
		timer.Reset(time.Hour)
	}
}
