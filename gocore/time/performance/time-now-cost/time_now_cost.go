package timenowcost

import "time"

// CallTimeNow calls time.Now() and returns the result.
func CallTimeNow() time.Time {
	return time.Now()
}

// CallTimeSince measures elapsed time since a starting point.
func CallTimeSince(start time.Time) time.Duration {
	return time.Since(start)
}

// CallTimeUnixNano returns the current Unix timestamp in nanoseconds.
func CallTimeUnixNano() int64 {
	return time.Now().UnixNano()
}
