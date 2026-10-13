package clockmodel

import (
	"testing"
	"time"
)

// mockUpZoneNames are the four world clocks shown in docs/theme.png, as the
// specification maps their labels to IANA names. The same four are the
// settings defaults in internal/settings; the duplication is deliberate
// (datum 0.7), because these pin the mock-up instant's expected output and
// must not move if the defaults ever change.
var mockUpZoneNames = []struct {
	label    string
	ianaName string
}{
	{label: "UTC", ianaName: "UTC"},
	{label: "LDN", ianaName: "Europe/London"},
	{label: "BER", ianaName: "Europe/Berlin"},
	{label: "SAO", ianaName: "America/Sao_Paulo"},
}

func loadMockUpZones(t *testing.T) []Zone {
	t.Helper()
	zones := make([]Zone, 0, len(mockUpZoneNames))
	for _, zoneName := range mockUpZoneNames {
		zones = append(zones, loadZoneOrFail(t, zoneName.label, zoneName.ianaName))
	}
	return zones
}

func loadZoneOrFail(t *testing.T, label string, ianaName string) Zone {
	t.Helper()
	zone, loadError := LoadZone(label, ianaName)
	if loadError != nil {
		t.Fatalf("loading fixture zone '%s' (%s): %v", label, ianaName, loadError)
	}
	return zone
}

func loadLocationOrFail(t *testing.T, ianaName string) *time.Location {
	t.Helper()
	return loadZoneOrFail(t, "LOC", ianaName).Location
}

func parseInstantOrFail(t *testing.T, rfc3339Instant string) time.Time {
	t.Helper()
	instant, parseError := time.Parse(time.RFC3339Nano, rfc3339Instant)
	if parseError != nil {
		t.Fatalf("parsing fixture instant '%s': %v", rfc3339Instant, parseError)
	}
	return instant
}
