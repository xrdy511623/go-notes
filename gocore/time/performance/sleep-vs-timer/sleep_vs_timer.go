package sleepvstimer

import "time"

// UseSleep blocks using time.Sleep.
func UseSleep(d time.Duration) {
	time.Sleep(d)
}

// UseTimer blocks using time.NewTimer + channel receive.
func UseTimer(d time.Duration) {
	timer := time.NewTimer(d)
	<-timer.C
	// timer 到期后无需 Stop
}

// UseTimerWithStop creates a Timer, immediately stops it.
// Measures the overhead of Timer lifecycle without actual sleep.
func UseTimerWithStop(d time.Duration) {
	timer := time.NewTimer(d)
	timer.Stop()
}
