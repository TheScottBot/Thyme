package clockmodel

import (
	"strconv"
	"time"
)

// The spoken form's words, English only by decision WD7. Indexed by
// time.Weekday and by time.Month minus one.
var (
	fullWeekdayNames = [7]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	fullMonthNames   = [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
)

// SpokenTime is the time and date as a screen reader should say them, in UK
// order: hours and minutes, then weekday, day, month and year. The shell sets
// it as the window's text each minute (WD9), since the widget takes no focus
// and has no other way to be read.
func SpokenTime(instant time.Time, localLocation *time.Location) string {
	localInstant := instant.In(localLocation)
	return localInstant.Format(timeWithoutSecondsLayout) + ", " +
		fullWeekdayNames[localInstant.Weekday()] + " " +
		strconv.Itoa(localInstant.Day()) + " " +
		fullMonthNames[localInstant.Month()-1] + " " +
		strconv.Itoa(localInstant.Year())
}
