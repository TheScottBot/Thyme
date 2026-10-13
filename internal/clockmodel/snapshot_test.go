package clockmodel

import (
	"reflect"
	"testing"
)

func TestTakeSnapshot_MockUpInstantMatchesTheme(t *testing.T) {
	localLocation := loadLocationOrFail(t, "Europe/London")
	instant := parseInstantOrFail(t, "2026-09-06T21:04:01+01:00")

	snapshot := TakeSnapshot(instant, localLocation, loadMockUpZones(t), true)

	expected := Snapshot{
		Time:     "21:04:01",
		DateLine: "sun 2026-09-06",
		ZoneTimes: []ZoneTime{
			{Label: "UTC", Time: "20:04"},
			{Label: "LDN", Time: "21:04"},
			{Label: "BER", Time: "22:04"},
			{Label: "SAO", Time: "17:04"},
		},
	}
	if !reflect.DeepEqual(snapshot, expected) {
		t.Errorf("expected %+v, got %+v", expected, snapshot)
	}
}

func TestTakeSnapshot_SecondsOffShowsHoursAndMinutesOnly(t *testing.T) {
	localLocation := loadLocationOrFail(t, "Europe/London")
	instant := parseInstantOrFail(t, "2026-09-06T21:04:01+01:00")

	snapshot := TakeSnapshot(instant, localLocation, loadMockUpZones(t), false)

	if snapshot.Time != "21:04" {
		t.Errorf("expected '21:04', got '%s'", snapshot.Time)
	}
	if snapshot.DateLine != "sun 2026-09-06" {
		t.Errorf("expected the date line unchanged, got '%s'", snapshot.DateLine)
	}
}

func TestTakeSnapshot_DateLineFollowsLocalZoneNotUtc(t *testing.T) {
	localLocation := loadLocationOrFail(t, "Europe/London")
	instant := parseInstantOrFail(t, "2026-09-06T23:30:00Z")

	snapshot := TakeSnapshot(instant, localLocation, nil, true)

	if snapshot.DateLine != "mon 2026-09-07" || snapshot.Time != "00:30:00" {
		t.Errorf("expected local 00:30:00 on mon 2026-09-07, got '%s' on '%s'", snapshot.Time, snapshot.DateLine)
	}
}

func TestTakeSnapshot_DateLineNamesEveryWeekdayInLowerCase(t *testing.T) {
	localLocation := loadLocationOrFail(t, "UTC")
	expectedDateLines := []string{
		"mon 2026-09-07", "tue 2026-09-08", "wed 2026-09-09", "thu 2026-09-10",
		"fri 2026-09-11", "sat 2026-09-12", "sun 2026-09-13",
	}
	firstInstant := parseInstantOrFail(t, "2026-09-07T12:00:00Z")

	for dayOffset, expectedDateLine := range expectedDateLines {
		instant := firstInstant.AddDate(0, 0, dayOffset)
		snapshot := TakeSnapshot(instant, localLocation, nil, true)
		if snapshot.DateLine != expectedDateLine {
			t.Errorf("expected '%s', got '%s'", expectedDateLine, snapshot.DateLine)
		}
	}
}

func TestTakeSnapshot_NoZonesGivesNoZoneTimes(t *testing.T) {
	localLocation := loadLocationOrFail(t, "UTC")
	instant := parseInstantOrFail(t, "2026-09-06T20:04:01Z")

	snapshot := TakeSnapshot(instant, localLocation, nil, true)

	if len(snapshot.ZoneTimes) != 0 {
		t.Errorf("expected no zone times, got %+v", snapshot.ZoneTimes)
	}
}

// Each case straddles a real transition by one minute either side. The
// expected wall times are written out by hand, not computed from tzdata, so a
// fixed offset implementation cannot pass.
func TestTakeSnapshot_ZoneTimeFollowsDaylightSavingTransition(t *testing.T) {
	testCases := []struct {
		name               string
		ianaName           string
		instantBefore      string
		expectedTimeBefore string
		instantAfter       string
		expectedTimeAfter  string
	}{
		{
			name:          "London springs forward at 01:00 UTC on 29 March 2026",
			ianaName:      "Europe/London",
			instantBefore: "2026-03-29T00:59:00Z", expectedTimeBefore: "00:59",
			instantAfter: "2026-03-29T01:01:00Z", expectedTimeAfter: "02:01",
		},
		{
			name:          "London falls back at 01:00 UTC on 25 October 2026",
			ianaName:      "Europe/London",
			instantBefore: "2026-10-25T00:59:00Z", expectedTimeBefore: "01:59",
			instantAfter: "2026-10-25T01:01:00Z", expectedTimeAfter: "01:01",
		},
		{
			name:          "Berlin springs forward at 01:00 UTC on 29 March 2026",
			ianaName:      "Europe/Berlin",
			instantBefore: "2026-03-29T00:59:00Z", expectedTimeBefore: "01:59",
			instantAfter: "2026-03-29T01:01:00Z", expectedTimeAfter: "03:01",
		},
		{
			name:          "Berlin falls back at 01:00 UTC on 25 October 2026",
			ianaName:      "Europe/Berlin",
			instantBefore: "2026-10-25T00:59:00Z", expectedTimeBefore: "02:59",
			instantAfter: "2026-10-25T01:01:00Z", expectedTimeAfter: "02:01",
		},
		{
			name:          "Sao Paulo began its last summer time at 03:00 UTC on 4 November 2018",
			ianaName:      "America/Sao_Paulo",
			instantBefore: "2018-11-04T02:59:00Z", expectedTimeBefore: "23:59",
			instantAfter: "2018-11-04T03:01:00Z", expectedTimeAfter: "01:01",
		},
		{
			name:          "UTC does not move across London's transition",
			ianaName:      "UTC",
			instantBefore: "2026-03-29T00:59:00Z", expectedTimeBefore: "00:59",
			instantAfter: "2026-03-29T01:01:00Z", expectedTimeAfter: "01:01",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			zones := []Zone{loadZoneOrFail(t, "TST", testCase.ianaName)}
			localLocation := loadLocationOrFail(t, "UTC")

			snapshotBefore := TakeSnapshot(parseInstantOrFail(t, testCase.instantBefore), localLocation, zones, true)
			snapshotAfter := TakeSnapshot(parseInstantOrFail(t, testCase.instantAfter), localLocation, zones, true)

			if len(snapshotBefore.ZoneTimes) != 1 || len(snapshotAfter.ZoneTimes) != 1 {
				t.Fatalf("expected one zone time either side, got %+v and %+v", snapshotBefore.ZoneTimes, snapshotAfter.ZoneTimes)
			}
			if snapshotBefore.ZoneTimes[0].Time != testCase.expectedTimeBefore {
				t.Errorf("before: expected '%s', got '%s'", testCase.expectedTimeBefore, snapshotBefore.ZoneTimes[0].Time)
			}
			if snapshotAfter.ZoneTimes[0].Time != testCase.expectedTimeAfter {
				t.Errorf("after: expected '%s', got '%s'", testCase.expectedTimeAfter, snapshotAfter.ZoneTimes[0].Time)
			}
		})
	}
}

func TestTakeSnapshot_LocalTimeFollowsDaylightSavingTransition(t *testing.T) {
	localLocation := loadLocationOrFail(t, "Europe/London")

	snapshotBefore := TakeSnapshot(parseInstantOrFail(t, "2026-03-29T00:59:59Z"), localLocation, nil, true)
	snapshotAfter := TakeSnapshot(parseInstantOrFail(t, "2026-03-29T01:00:00Z"), localLocation, nil, true)

	if snapshotBefore.Time != "00:59:59" || snapshotAfter.Time != "02:00:00" {
		t.Errorf("expected 00:59:59 then 02:00:00, got '%s' then '%s'", snapshotBefore.Time, snapshotAfter.Time)
	}
}
