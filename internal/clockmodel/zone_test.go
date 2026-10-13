package clockmodel

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadZone_LoadsEveryMockUpZone(t *testing.T) {
	for _, zoneName := range mockUpZoneNames {
		zone, loadError := LoadZone(zoneName.label, zoneName.ianaName)
		if loadError != nil {
			t.Errorf("'%s' (%s): %v", zoneName.label, zoneName.ianaName, loadError)
			continue
		}
		if zone.Label != zoneName.label || zone.Location == nil || zone.Location.String() != zoneName.ianaName {
			t.Errorf("expected label '%s' at %s, got %+v", zoneName.label, zoneName.ianaName, zone)
		}
	}
}

func TestLoadZone_RejectsWithErrorNamingTheLabel(t *testing.T) {
	testCases := []struct {
		name           string
		ianaName       string
		expectedReason ZoneLoadFailureReason
	}{
		{name: "an unknown IANA name", ianaName: "Mars/Olympus_Mons", expectedReason: ReasonUnknownZoneName},
		{name: "an empty name, which the standard library would load as UTC", ianaName: "", expectedReason: ReasonEmptyZoneName},
		{name: "Local, which would follow the host rather than name a zone", ianaName: "Local", expectedReason: ReasonHostDependentZoneName},
		{name: "a path rather than a name", ianaName: "../Europe/London", expectedReason: ReasonUnknownZoneName},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			zone, loadError := LoadZone("XYZ", testCase.ianaName)

			if loadError == nil {
				t.Fatalf("expected an error, got %+v", zone)
			}
			if !strings.Contains(loadError.Error(), "'XYZ'") {
				t.Errorf("expected the error to name the label, got '%v'", loadError)
			}
			var zoneLoadError *ZoneLoadError
			if !errors.As(loadError, &zoneLoadError) {
				t.Fatalf("expected a *ZoneLoadError, got %T", loadError)
			}
			if zoneLoadError.Reason != testCase.expectedReason {
				t.Errorf("expected reason '%s', got '%s'", testCase.expectedReason, zoneLoadError.Reason)
			}
			if zone.Location != nil {
				t.Errorf("expected no location on failure, got %s", zone.Location)
			}
		})
	}
}
