package clockmodel

import "time"

// WakeMargin is how long after a wall clock boundary the widget wakes. Waking
// exactly on the boundary is unsafe: the timer's resolution is the clock
// interrupt period, so a wake aimed at the boundary can fire a little early,
// read the old second, paint it twice, and then skip the next one. A skipped
// second is visible to anyone watching; 20 ms of lateness is not.
const WakeMargin = 20 * time.Millisecond

// NextWake returns how long to wait from now until just after the next whole
// second, or whole minute when seconds are hidden, of now's wall clock. It is
// recomputed from a fresh reading after every wake rather than run as a fixed
// period ticker, so drift and wall clock changes never accumulate.
//
// An instant exactly on a boundary has already been painted for that
// boundary, so the next one is a full period away.
func NextWake(now time.Time, showSeconds bool) time.Duration {
	period := time.Second
	elapsedInPeriod := time.Duration(now.Nanosecond())
	if !showSeconds {
		period = time.Minute
		elapsedInPeriod += time.Duration(now.Second()) * time.Second
	}
	return period - elapsedInPeriod + WakeMargin
}
