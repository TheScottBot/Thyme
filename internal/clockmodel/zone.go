package clockmodel

import (
	"fmt"
	"time"

	// Embedded so zones load on a machine without Go installed: Windows has no
	// zoneinfo files for the time package to read. Measured in WE2 at about
	// 400 KB of the exe; see docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md.
	_ "time/tzdata"
)

// Zone is one world clock: the label shown beside it and where it is.
type Zone struct {
	Label    string
	Location *time.Location
}

// ZoneLoadFailureReason is the category a failed zone load is counted and
// logged under.
type ZoneLoadFailureReason string

const (
	ReasonEmptyZoneName         ZoneLoadFailureReason = "empty zone name"
	ReasonHostDependentZoneName ZoneLoadFailureReason = "host dependent zone name"
	ReasonUnknownZoneName       ZoneLoadFailureReason = "unknown zone name"
)

// hostDependentZoneName is accepted by the standard library but names whatever
// zone the host is set to, which is the local clock's job, not a world clock's.
const hostDependentZoneName = "Local"

// ZoneLoadError names the label of the zone that failed, because the label is
// what the user sees and edits; the IANA name alone would not say which line of
// the settings file is wrong.
type ZoneLoadError struct {
	Label    string
	IanaName string
	Reason   ZoneLoadFailureReason
	Cause    error
}

func (zoneLoadError *ZoneLoadError) Error() string {
	if zoneLoadError.Cause != nil {
		return fmt.Sprintf("zone '%s': %s '%s': %v", zoneLoadError.Label, zoneLoadError.Reason, zoneLoadError.IanaName, zoneLoadError.Cause)
	}
	return fmt.Sprintf("zone '%s': %s '%s'", zoneLoadError.Label, zoneLoadError.Reason, zoneLoadError.IanaName)
}

func (zoneLoadError *ZoneLoadError) Unwrap() error {
	return zoneLoadError.Cause
}

// LoadZone loads one world clock or fails loudly. It never falls back: the
// standard library quietly maps an empty name to UTC, and a nil location
// prints as UTC, so a fallback would show a plausible wrong time with nothing
// to say it was wrong.
func LoadZone(label string, ianaName string) (Zone, error) {
	if ianaName == "" {
		return Zone{}, &ZoneLoadError{Label: label, IanaName: ianaName, Reason: ReasonEmptyZoneName}
	}
	if ianaName == hostDependentZoneName {
		return Zone{}, &ZoneLoadError{Label: label, IanaName: ianaName, Reason: ReasonHostDependentZoneName}
	}
	location, loadError := time.LoadLocation(ianaName)
	if loadError != nil {
		return Zone{}, &ZoneLoadError{Label: label, IanaName: ianaName, Reason: ReasonUnknownZoneName, Cause: loadError}
	}
	return Zone{Label: label, Location: location}, nil
}
