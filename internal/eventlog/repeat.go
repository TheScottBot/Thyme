package eventlog

import "math/bits"

// IsWorthLogging is true for the first occurrence of a repeating fault and
// then each time its count doubles, so a fault that happens every tick is
// visible in the log without flooding it.
func IsWorthLogging(count int) bool {
	return count > 0 && bits.OnesCount(uint(count)) == 1
}
