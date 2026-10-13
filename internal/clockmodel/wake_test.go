package clockmodel

import (
	"testing"
	"time"
)

func TestNextWake_LandsAfterBoundaryByExactlyTheMargin(t *testing.T) {
	testCases := []struct {
		name                 string
		now                  string
		showSeconds          bool
		expectedNextBoundary string
	}{
		{name: "seconds on, exactly on a second", now: "2026-09-06T21:04:01.000Z", showSeconds: true, expectedNextBoundary: "2026-09-06T21:04:02Z"},
		{name: "seconds on, just before a second", now: "2026-09-06T21:04:01.999Z", showSeconds: true, expectedNextBoundary: "2026-09-06T21:04:02Z"},
		{name: "seconds on, inside the margin of the next second", now: "2026-09-06T21:04:01.980Z", showSeconds: true, expectedNextBoundary: "2026-09-06T21:04:02Z"},
		{name: "seconds off, exactly on a second", now: "2026-09-06T21:04:01.000Z", showSeconds: false, expectedNextBoundary: "2026-09-06T21:05:00Z"},
		{name: "seconds off, just before a second", now: "2026-09-06T21:04:01.999Z", showSeconds: false, expectedNextBoundary: "2026-09-06T21:05:00Z"},
		{name: "seconds off, inside the margin of the next second", now: "2026-09-06T21:04:01.980Z", showSeconds: false, expectedNextBoundary: "2026-09-06T21:05:00Z"},
		{name: "seconds off, exactly on a minute", now: "2026-09-06T21:04:00.000Z", showSeconds: false, expectedNextBoundary: "2026-09-06T21:05:00Z"},
		{name: "seconds off, last instant of a minute", now: "2026-09-06T21:04:59.999Z", showSeconds: false, expectedNextBoundary: "2026-09-06T21:05:00Z"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			now := parseInstantOrFail(t, testCase.now)
			expectedNextBoundary := parseInstantOrFail(t, testCase.expectedNextBoundary)

			wakeDelay := NextWake(now, testCase.showSeconds)

			wakeInstant := now.Add(wakeDelay)
			if !wakeInstant.Equal(expectedNextBoundary.Add(WakeMargin)) {
				t.Errorf("expected wake at %s plus the margin, got %s", expectedNextBoundary.Format(time.RFC3339Nano), wakeInstant.Format(time.RFC3339Nano))
			}
			if !expectedNextBoundary.After(now) {
				t.Errorf("fixture error: boundary %s is not strictly after now %s", testCase.expectedNextBoundary, testCase.now)
			}
		})
	}
}

func TestWakeMargin_IsTwentyMilliseconds(t *testing.T) {
	if WakeMargin != 20*time.Millisecond {
		t.Errorf("expected the specification's 20 ms margin, got %s", WakeMargin)
	}
}
