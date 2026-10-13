package eventlog

import "testing"

// A fault that repeats every tick is logged the first time and then each
// time its count doubles, so it stays visible without flooding the log.
func TestIsWorthLogging_LogsTheFirstAndThenEachDoubling(t *testing.T) {
	var logged []int
	for count := 0; count <= 70; count++ {
		if IsWorthLogging(count) {
			logged = append(logged, count)
		}
	}
	expected := []int{1, 2, 4, 8, 16, 32, 64}
	if len(logged) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, logged)
	}
	for index := range expected {
		if logged[index] != expected[index] {
			t.Errorf("expected %v, got %v", expected, logged)
		}
	}
}
