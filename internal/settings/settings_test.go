package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDefaults_MatchTheSpecificationAndTheMockUp(t *testing.T) {
	expected := Settings{
		Position:     Position{X: 120, Y: 120},
		ScalePercent: 100,
		AlwaysOnTop:  false,
		ShowInTray:   true,
		ShowSeconds:  true,
		Zones: []ZoneSetting{
			{Label: "UTC", TimeZone: "UTC"},
			{Label: "LDN", TimeZone: "Europe/London"},
			{Label: "BER", TimeZone: "Europe/Berlin"},
			{Label: "SAO", TimeZone: "America/Sao_Paulo"},
		},
	}

	if defaults := Defaults(); !reflect.DeepEqual(defaults, expected) {
		t.Errorf("expected %+v, got %+v", expected, defaults)
	}
}

func TestDefaults_PassValidation(t *testing.T) {
	if rejections := Validate(Defaults()); len(rejections) != 0 {
		t.Errorf("expected defaults to be valid, got %+v", rejections)
	}
}

func TestDefaults_ReturnsIndependentZoneSlices(t *testing.T) {
	first := Defaults()
	if len(first.Zones) == 0 {
		t.Fatal("expected default zones")
	}
	first.Zones[0].Label = "XXX"

	if Defaults().Zones[0].Label != "UTC" {
		t.Error("expected a change to one copy of the defaults not to reach the next")
	}
}

func TestScaleStepsPercent_AreTheDecidedSteps(t *testing.T) {
	if !reflect.DeepEqual(ScaleStepsPercent(), []int{75, 100, 125, 150, 200}) {
		t.Errorf("expected the WD5 steps, got %v", ScaleStepsPercent())
	}
}

func TestRelativeFilePath_IsDerivedFromTheProductName(t *testing.T) {
	if RelativeFilePath() != filepath.Join("Thyme", "settings.json") {
		t.Errorf("expected Thyme/settings.json, got '%s'", RelativeFilePath())
	}
}

func TestEncodeThenParse_RoundTripsEveryField(t *testing.T) {
	original := everyFieldChanged()

	encoded, encodeError := Encode(original)
	if encodeError != nil {
		t.Fatalf("unexpected error: %v", encodeError)
	}
	parsed, report := Parse(encoded)

	if !reflect.DeepEqual(parsed, original) {
		t.Errorf("expected %+v, got %+v", original, parsed)
	}
	if len(report.Rejections) != 0 || report.Source != SourceFile || !report.OverwriteAllowed {
		t.Errorf("expected a clean report from the file, got %+v", report)
	}
}

func TestEncodeThenParse_RoundTripsZeroZones(t *testing.T) {
	original := everyFieldChanged()
	original.Zones = []ZoneSetting{}

	encoded, encodeError := Encode(original)
	if encodeError != nil {
		t.Fatalf("unexpected error: %v", encodeError)
	}
	parsed, report := Parse(encoded)

	if len(parsed.Zones) != 0 || len(report.Rejections) != 0 {
		t.Errorf("expected zero zones and no rejections, got %+v and %+v", parsed.Zones, report.Rejections)
	}
}

func TestEncode_WritesVersionAndFieldsInTableOrder(t *testing.T) {
	encoded, encodeError := Encode(Defaults())
	if encodeError != nil {
		t.Fatalf("unexpected error: %v", encodeError)
	}

	expectedOrder := []string{`"version": 1`, `"position"`, `"scalePercent"`, `"alwaysOnTop"`, `"showInTray"`, `"showSeconds"`, `"zones"`}
	previousIndex := -1
	for _, expectedFragment := range expectedOrder {
		fragmentIndex := strings.Index(string(encoded), expectedFragment)
		if fragmentIndex <= previousIndex {
			t.Fatalf("expected '%s' after the previous field in:\n%s", expectedFragment, encoded)
		}
		previousIndex = fragmentIndex
	}
}

func TestEncode_RefusesInvalidSettings(t *testing.T) {
	invalid := Defaults()
	invalid.ScalePercent = 101

	if _, encodeError := Encode(invalid); encodeError == nil {
		t.Error("expected settings that would not load back to be refused")
	}
}

func TestLoad_MissingFileYieldsDefaultsWithNoRejections(t *testing.T) {
	loaded, report := Load(filepath.Join(t.TempDir(), "settings.json"))

	if !reflect.DeepEqual(loaded, Defaults()) {
		t.Errorf("expected defaults, got %+v", loaded)
	}
	if len(report.Rejections) != 0 || report.Source != SourceDefaults || !report.OverwriteAllowed {
		t.Errorf("expected a clean report naming defaults, got %+v", report)
	}
}

func TestLoad_ReadsFileExactlyAtTheBound(t *testing.T) {
	document := validDocumentWith(t, "scalePercent", "150")
	padded := document + strings.Repeat(" ", MaximumFileBytes-len(document))
	settingsPath := writeSettingsFile(t, padded)

	loaded, report := Load(settingsPath)

	if len(report.Rejections) != 0 || loaded.ScalePercent != 150 {
		t.Errorf("expected a file of exactly %d bytes to load, got %+v", MaximumFileBytes, report)
	}
}

// The oversized file is a valid document padded with spaces, so the only way
// it can be refused is by the size bound, before any parsing happens.
func TestLoad_RefusesFileOneByteOverTheBoundBeforeParsing(t *testing.T) {
	document := validDocumentWith(t, "scalePercent", "150")
	padded := document + strings.Repeat(" ", MaximumFileBytes+1-len(document))
	settingsPath := writeSettingsFile(t, padded)

	loaded, report := Load(settingsPath)

	if !reflect.DeepEqual(loaded, Defaults()) {
		t.Errorf("expected defaults, got %+v", loaded)
	}
	if !hasRejection(report, "", ReasonFileTooLarge) || report.Source != SourceDefaults {
		t.Errorf("expected the file to be refused as too large, got %+v", report)
	}
}

func TestLoad_UnreadablePathYieldsDefaultsAndCountsIt(t *testing.T) {
	directoryInPlaceOfFile := t.TempDir()

	loaded, report := Load(directoryInPlaceOfFile)

	if !reflect.DeepEqual(loaded, Defaults()) || !hasRejection(report, "", ReasonFileUnreadable) {
		t.Errorf("expected defaults and an unreadable file rejection, got %+v", report)
	}
}

func TestParse_SpecificationRejectionsHaveDistinctReasons(t *testing.T) {
	testCases := []struct {
		name           string
		document       func(t *testing.T) string
		expectedField  string
		expectedReason RejectionReason
	}{
		{
			name: "an unknown field",
			document: func(t *testing.T) string {
				return strings.Replace(validDocumentWith(t, "scalePercent", "125"), "{", `{"colour":"red",`, 1)
			},
			expectedField: "", expectedReason: ReasonUnknownField,
		},
		{
			name: "a fifth zone",
			document: func(t *testing.T) string {
				return validDocumentWith(t, "zones", `[
					{"label":"A","timeZone":"UTC"},{"label":"B","timeZone":"UTC"},
					{"label":"C","timeZone":"UTC"},{"label":"D","timeZone":"UTC"},
					{"label":"E","timeZone":"UTC"}]`)
			},
			expectedField: "zones", expectedReason: ReasonTooManyZones,
		},
		{
			name: "a four character label",
			document: func(t *testing.T) string {
				return validDocumentWith(t, "zones", `[{"label":"LOND","timeZone":"Europe/London"}]`)
			},
			expectedField: "zones", expectedReason: ReasonLabelLength,
		},
		{
			name: "an empty IANA name",
			document: func(t *testing.T) string {
				return validDocumentWith(t, "zones", `[{"label":"LDN","timeZone":""}]`)
			},
			expectedField: "zones", expectedReason: ReasonEmptyTimeZone,
		},
	}
	seenReasons := map[RejectionReason]string{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed, report := Parse([]byte(testCase.document(t)))

			if !hasRejection(report, testCase.expectedField, testCase.expectedReason) {
				t.Errorf("expected rejection %q of field %q, got %+v", testCase.expectedReason, testCase.expectedField, report.Rejections)
			}
			if len(report.Rejections) != 1 {
				t.Errorf("expected exactly one rejection, got %+v", report.Rejections)
			}
			if parsed.ScalePercent != 125 {
				t.Errorf("expected the other fields to be kept, got scale %d", parsed.ScalePercent)
			}
			if previousCase, seen := seenReasons[testCase.expectedReason]; seen {
				t.Errorf("reason %q is shared with case %q", testCase.expectedReason, previousCase)
			}
			seenReasons[testCase.expectedReason] = testCase.name
		})
	}
}

func TestParse_RejectedFieldFallsBackToItsDefaultOnly(t *testing.T) {
	document := validDocumentWith(t, "zones", `[{"label":"LOND","timeZone":"Europe/London"}]`)

	parsed, _ := Parse([]byte(document))

	if !reflect.DeepEqual(parsed.Zones, Defaults().Zones) {
		t.Errorf("expected the default zones, got %+v", parsed.Zones)
	}
	if parsed.Position != (Position{X: 10, Y: 20}) || parsed.ScalePercent != 125 || !parsed.AlwaysOnTop || parsed.ShowSeconds {
		t.Errorf("expected every other field from the file, got %+v", parsed)
	}
}

func TestParse_RejectsEachFieldLevelProblem(t *testing.T) {
	testCases := []struct {
		name           string
		fieldName      string
		replacement    string
		expectedReason RejectionReason
	}{
		{name: "null is not a boolean", fieldName: "alwaysOnTop", replacement: "null", expectedReason: ReasonWrongType},
		{name: "a string is not a boolean", fieldName: "showInTray", replacement: `"yes"`, expectedReason: ReasonWrongType},
		{name: "a fraction is not a scale", fieldName: "scalePercent", replacement: "100.5", expectedReason: ReasonWrongType},
		{name: "a scale between steps", fieldName: "scalePercent", replacement: "110", expectedReason: ReasonOutOfRange},
		{name: "a position beyond 32 bits", fieldName: "position", replacement: `{"x": 2147483648, "y": 0}`, expectedReason: ReasonOutOfRange},
		{name: "a fractional position", fieldName: "position", replacement: `{"x": 1.5, "y": 0}`, expectedReason: ReasonWrongType},
		{name: "a position missing y", fieldName: "position", replacement: `{"x": 1}`, expectedReason: ReasonMissingField},
		{name: "a position with an extra member", fieldName: "position", replacement: `{"x": 1, "y": 2, "z": 3}`, expectedReason: ReasonUnknownField},
		{name: "a position given twice over", fieldName: "position", replacement: `{"x": 1, "x": 2, "y": 3}`, expectedReason: ReasonDuplicateField},
		{name: "zones that are not a list", fieldName: "zones", replacement: `{"label":"LDN"}`, expectedReason: ReasonWrongType},
		{name: "a null zones list", fieldName: "zones", replacement: "null", expectedReason: ReasonWrongType},
		{name: "an empty label", fieldName: "zones", replacement: `[{"label":"","timeZone":"UTC"}]`, expectedReason: ReasonLabelLength},
		{name: "a label with a control character", fieldName: "zones", replacement: `[{"label":"L\u0007N","timeZone":"UTC"}]`, expectedReason: ReasonLabelNotPrintable},
		{name: "a time zone name over the bound", fieldName: "zones", replacement: `[{"label":"LNG","timeZone":"` + strings.Repeat("A", MaximumTimeZoneNameBytes+1) + `"}]`, expectedReason: ReasonTimeZoneTooLong},
		{name: "a zone missing its time zone", fieldName: "zones", replacement: `[{"label":"LDN"}]`, expectedReason: ReasonMissingField},
		{name: "a zone with an extra member", fieldName: "zones", replacement: `[{"label":"LDN","timeZone":"UTC","colour":"red"}]`, expectedReason: ReasonUnknownField},
		{name: "a zone that is not an object", fieldName: "zones", replacement: `["LDN"]`, expectedReason: ReasonWrongType},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed, report := Parse([]byte(validDocumentWith(t, testCase.fieldName, testCase.replacement)))

			if !hasRejection(report, testCase.fieldName, testCase.expectedReason) || len(report.Rejections) != 1 {
				t.Errorf("expected only %q of %q, got %+v", testCase.expectedReason, testCase.fieldName, report.Rejections)
			}
			if rejections := Validate(parsed); len(rejections) != 0 {
				t.Errorf("expected the result to be valid after fallback, got %+v", rejections)
			}
		})
	}
}

func TestParse_AcceptsThreeCharacterLabelOutsideAscii(t *testing.T) {
	parsed, report := Parse([]byte(validDocumentWith(t, "zones", `[{"label":"SÃO","timeZone":"America/Sao_Paulo"}]`)))

	if len(report.Rejections) != 0 || len(parsed.Zones) != 1 || parsed.Zones[0].Label != "SÃO" {
		t.Errorf("expected a three character label to be counted in characters not bytes, got %+v", report.Rejections)
	}
}

func TestParse_MissingFieldFallsBackAndIsCounted(t *testing.T) {
	document := strings.Replace(validDocumentWith(t, "scalePercent", "125"), `"alwaysOnTop":true,`, "", 1)

	parsed, report := Parse([]byte(document))

	if !hasRejection(report, "alwaysOnTop", ReasonMissingField) || parsed.AlwaysOnTop != Defaults().AlwaysOnTop {
		t.Errorf("expected a counted missing field and its default, got %+v", report.Rejections)
	}
}

func TestParse_DuplicateTopLevelFieldFallsBack(t *testing.T) {
	document := strings.Replace(validDocumentWith(t, "scalePercent", "125"), "{", `{"scalePercent":150,`, 1)

	parsed, report := Parse([]byte(document))

	if !hasRejection(report, "scalePercent", ReasonDuplicateField) || parsed.ScalePercent != Defaults().ScalePercent {
		t.Errorf("expected the duplicated field to fall back rather than pick a winner, got %+v and scale %d", report.Rejections, parsed.ScalePercent)
	}
}

func TestParse_WholeDocumentProblemsYieldDefaults(t *testing.T) {
	testCases := []struct {
		name           string
		document       string
		expectedReason RejectionReason
	}{
		{name: "not JSON", document: "scale = 125", expectedReason: ReasonMalformedDocument},
		{name: "a list, not an object", document: "[1, 2]", expectedReason: ReasonMalformedDocument},
		{name: "an object followed by more", document: `{"version": 1} {}`, expectedReason: ReasonMalformedDocument},
		{name: "a truncated object", document: `{"version": 1, "position": {"x": 1`, expectedReason: ReasonMalformedDocument},
		{name: "empty", document: "", expectedReason: ReasonMalformedDocument},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed, report := Parse([]byte(testCase.document))

			if !reflect.DeepEqual(parsed, Defaults()) || report.Source != SourceDefaults {
				t.Errorf("expected defaults from defaults, got %+v from %s", parsed, report.Source)
			}
			if !hasRejection(report, "", testCase.expectedReason) || len(report.Rejections) != 1 {
				t.Errorf("expected only %q, got %+v", testCase.expectedReason, report.Rejections)
			}
		})
	}
}

func TestParse_VersionProblemsYieldDefaultsAndForbidOverwriting(t *testing.T) {
	testCases := []struct {
		name           string
		document       func(t *testing.T) string
		expectedReason RejectionReason
	}{
		{
			name: "no version",
			document: func(t *testing.T) string {
				return strings.Replace(validDocumentWith(t, "version", "1"), `"version":1`, `"versionless":1`, 1)
			},
			expectedReason: ReasonVersionMissing,
		},
		{
			name:           "a newer version",
			document:       func(t *testing.T) string { return validDocumentWith(t, "version", "2") },
			expectedReason: ReasonVersionUnsupported,
		},
		{
			name:           "a version written as a string",
			document:       func(t *testing.T) string { return validDocumentWith(t, "version", `"1"`) },
			expectedReason: ReasonVersionUnsupported,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsed, report := Parse([]byte(testCase.document(t)))

			if !reflect.DeepEqual(parsed, Defaults()) || report.Source != SourceDefaults {
				t.Errorf("expected defaults, got %+v from %s", parsed, report.Source)
			}
			if !hasRejection(report, "version", testCase.expectedReason) || len(report.Rejections) != 1 {
				t.Errorf("expected only %q, got %+v", testCase.expectedReason, report.Rejections)
			}
			if report.OverwriteAllowed {
				t.Error("expected a file of an unknown version to be protected from being saved over")
			}
		})
	}
}

// Datum 0.15: a rejection carries a field name from the contract and a reason,
// never anything read from the file, since a hand edited file is the user's own.
func TestParse_RejectionsNeverCarryFileContents(t *testing.T) {
	document := strings.Replace(validDocumentWith(t, "zones", `[{"label":"S3CR3T","timeZone":"Private/Place"}]`), "{", `{"myHomeAddress":"1 High Street",`, 1)

	_, report := Parse([]byte(document))

	for _, rejection := range report.Rejections {
		for _, fromFile := range []string{"S3CR3T", "Private", "myHomeAddress", "High Street"} {
			if strings.Contains(rejection.Field, fromFile) || strings.Contains(string(rejection.Reason), fromFile) {
				t.Errorf("rejection %+v carries file contents", rejection)
			}
		}
	}
	if len(report.Rejections) != 2 {
		t.Errorf("expected the unknown field and the zones to be rejected, got %+v", report.Rejections)
	}
}

func TestSave_WritesFileThatLoadsBack(t *testing.T) {
	settingsPath := filepath.Join(t.TempDir(), "Thyme", "settings.json")

	if saveError := Save(settingsPath, everyFieldChanged()); saveError != nil {
		t.Fatalf("unexpected error: %v", saveError)
	}
	loaded, report := Load(settingsPath)

	if !reflect.DeepEqual(loaded, everyFieldChanged()) || len(report.Rejections) != 0 {
		t.Errorf("expected the saved settings back, got %+v with %+v", loaded, report.Rejections)
	}
}

func TestSave_ReplacesExistingFileAndLeavesNoTemporaryFile(t *testing.T) {
	settingsPath := writeSettingsFile(t, "an old hand edit")

	if saveError := Save(settingsPath, everyFieldChanged()); saveError != nil {
		t.Fatalf("unexpected error: %v", saveError)
	}

	directoryEntries, readError := os.ReadDir(filepath.Dir(settingsPath))
	if readError != nil {
		t.Fatalf("reading the settings folder: %v", readError)
	}
	if len(directoryEntries) != 1 || directoryEntries[0].Name() != "settings.json" {
		t.Errorf("expected only settings.json in the folder, got %v", directoryEntries)
	}
	if loaded, _ := Load(settingsPath); !reflect.DeepEqual(loaded, everyFieldChanged()) {
		t.Errorf("expected the old file to be replaced, got %+v", loaded)
	}
}

func TestSave_InvalidSettingsLeaveExistingFileUntouched(t *testing.T) {
	settingsPath := writeSettingsFile(t, "an old hand edit")
	invalid := everyFieldChanged()
	invalid.Zones = append(invalid.Zones, ZoneSetting{Label: "TOOLONG", TimeZone: "UTC"})

	if saveError := Save(settingsPath, invalid); saveError == nil {
		t.Fatal("expected invalid settings to be refused")
	}

	if fileContents, _ := os.ReadFile(settingsPath); string(fileContents) != "an old hand edit" {
		t.Errorf("expected the existing file untouched, got %q", fileContents)
	}
}

func TestSave_FailedRenameRemovesTemporaryFile(t *testing.T) {
	settingsFolder := t.TempDir()
	directoryInPlaceOfFile := filepath.Join(settingsFolder, "settings.json")
	if mkdirError := os.Mkdir(directoryInPlaceOfFile, 0o755); mkdirError != nil {
		t.Fatalf("creating fixture directory: %v", mkdirError)
	}
	if writeError := os.WriteFile(filepath.Join(directoryInPlaceOfFile, "keep"), []byte("x"), 0o644); writeError != nil {
		t.Fatalf("filling fixture directory: %v", writeError)
	}

	if saveError := Save(directoryInPlaceOfFile, Defaults()); saveError == nil {
		t.Fatal("expected renaming over a non empty directory to fail")
	}

	directoryEntries, readError := os.ReadDir(settingsFolder)
	if readError != nil {
		t.Fatalf("reading the settings folder: %v", readError)
	}
	if len(directoryEntries) != 1 {
		t.Errorf("expected the temporary file to be removed after the failure, got %v", directoryEntries)
	}
}

func TestLoad_EveryWholeFileRefusalForbidsOverwriting(t *testing.T) {
	testCases := []struct {
		name         string
		settingsPath func(t *testing.T) string
	}{
		{name: "too large", settingsPath: func(t *testing.T) string {
			return writeSettingsFile(t, strings.Repeat(" ", MaximumFileBytes+1))
		}},
		{name: "unreadable", settingsPath: func(t *testing.T) string { return t.TempDir() }},
		{name: "malformed", settingsPath: func(t *testing.T) string { return writeSettingsFile(t, "scale = 125") }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, report := Load(testCase.settingsPath(t))

			if report.OverwriteAllowed {
				t.Errorf("expected a file Thyme could not understand to be protected, got %+v", report)
			}
		})
	}
}
