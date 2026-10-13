package clockmodel

import "time"

// The displayed formats, each written once. The date line is a lower case
// three letter weekday then the ISO date, as the theme shows it.
const (
	timeWithSecondsLayout    = "15:04:05"
	timeWithoutSecondsLayout = "15:04"
	zoneTimeLayout           = "15:04"
	isoDateLayout            = "2006-01-02"
)

// lowerCaseWeekdayNames is indexed by time.Weekday. A table rather than
// lower casing the standard library's names, because the interface is English
// only by decision WD7 and this keeps the displayed words in one visible place.
var lowerCaseWeekdayNames = [7]string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// ZoneTime is one world clock's line: its label and its HH:MM.
type ZoneTime struct {
	Label string
	Time  string
}

// Snapshot is every string the widget shows for one instant.
type Snapshot struct {
	Time      string
	DateLine  string
	ZoneTimes []ZoneTime
}

// TakeSnapshot formats instant for display. The local location is passed in
// rather than read from time.Local so the result depends only on the
// arguments and can be tested in any zone on any host.
func TakeSnapshot(instant time.Time, localLocation *time.Location, zones []Zone, showSeconds bool) Snapshot {
	localInstant := instant.In(localLocation)

	timeLayout := timeWithoutSecondsLayout
	if showSeconds {
		timeLayout = timeWithSecondsLayout
	}

	zoneTimes := make([]ZoneTime, 0, len(zones))
	for _, zone := range zones {
		zoneTimes = append(zoneTimes, ZoneTime{Label: zone.Label, Time: instant.In(zone.Location).Format(zoneTimeLayout)})
	}

	return Snapshot{
		Time:      localInstant.Format(timeLayout),
		DateLine:  lowerCaseWeekdayNames[localInstant.Weekday()] + " " + localInstant.Format(isoDateLayout),
		ZoneTimes: zoneTimes,
	}
}
