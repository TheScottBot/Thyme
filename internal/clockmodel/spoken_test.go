package clockmodel

import "testing"

// WD9: the window's text is set to the time in words a screen reader can say,
// so assistive technology that reads the window name gets something useful.
func TestSpokenTime_GivesTheTimeAndTheFullDate(t *testing.T) {
	localLocation := loadLocationOrFail(t, "Europe/London")

	spoken := SpokenTime(parseInstantOrFail(t, "2026-09-06T21:04:01+01:00"), localLocation)

	if spoken != "21:04, Sunday 6 September 2026" {
		t.Errorf("expected the mock-up instant spoken in full, got %q", spoken)
	}
}

func TestSpokenTime_UsesTheLocalZone(t *testing.T) {
	localLocation := loadLocationOrFail(t, "Europe/London")

	spoken := SpokenTime(parseInstantOrFail(t, "2026-12-31T23:30:00Z"), localLocation)

	if spoken != "23:30, Thursday 31 December 2026" {
		t.Errorf("expected winter time in London, got %q", spoken)
	}
}
