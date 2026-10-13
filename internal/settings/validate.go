package settings

import (
	"slices"
	"unicode"
	"unicode/utf8"
)

// RejectionReason is the category a rejected setting is counted and logged
// under. Reasons and field names are the only things about a settings file
// that are ever logged, never its contents (datum 0.15).
type RejectionReason string

const (
	ReasonFileTooLarge       RejectionReason = "file too large"
	ReasonFileUnreadable     RejectionReason = "file unreadable"
	ReasonMalformedDocument  RejectionReason = "malformed document"
	ReasonVersionMissing     RejectionReason = "version missing"
	ReasonVersionUnsupported RejectionReason = "version unsupported"
	ReasonUnknownField       RejectionReason = "unknown field"
	ReasonDuplicateField     RejectionReason = "duplicate field"
	ReasonMissingField       RejectionReason = "missing field"
	ReasonWrongType          RejectionReason = "wrong type"
	ReasonOutOfRange         RejectionReason = "out of range"
	ReasonTooManyZones       RejectionReason = "too many zones"
	ReasonLabelLength        RejectionReason = "label length"
	ReasonLabelNotPrintable  RejectionReason = "label not printable"
	ReasonEmptyTimeZone      RejectionReason = "empty time zone"
	ReasonTimeZoneTooLong    RejectionReason = "time zone too long"
)

// Rejection is one refused setting. Field is a field name from the contract,
// or empty when the problem is the whole file or a field the contract does not
// have, since an unknown field's name is itself file contents.
type Rejection struct {
	Field  string
	Reason RejectionReason
}

// Validate applies every bound to settings that did not come from Parse, such
// as those the shell is about to save. Parse uses the same rules, so a
// setting Validate accepts always loads back.
func Validate(settings Settings) []Rejection {
	var rejections []Rejection
	if reason := validateScalePercent(settings.ScalePercent); reason != "" {
		rejections = append(rejections, Rejection{Field: fieldNameScalePercent, Reason: reason})
	}
	if reason := validateZones(settings.Zones); reason != "" {
		rejections = append(rejections, Rejection{Field: fieldNameZones, Reason: reason})
	}
	return rejections
}

func validateScalePercent(scalePercent int) RejectionReason {
	if !slices.Contains(scaleStepsPercent, scalePercent) {
		return ReasonOutOfRange
	}
	return ""
}

func validateZones(zones []ZoneSetting) RejectionReason {
	if len(zones) > MaximumZoneCount {
		return ReasonTooManyZones
	}
	for _, zone := range zones {
		if reason := validateLabel(zone.Label); reason != "" {
			return reason
		}
		if reason := validateTimeZoneName(zone.TimeZone); reason != "" {
			return reason
		}
	}
	return ""
}

func validateLabel(label string) RejectionReason {
	if !utf8.ValidString(label) {
		return ReasonLabelNotPrintable
	}
	characterCount := utf8.RuneCountInString(label)
	if characterCount < MinimumLabelCharacters || characterCount > MaximumLabelCharacters {
		return ReasonLabelLength
	}
	for _, character := range label {
		if !unicode.IsPrint(character) {
			return ReasonLabelNotPrintable
		}
	}
	return ""
}

func validateTimeZoneName(timeZoneName string) RejectionReason {
	if timeZoneName == "" {
		return ReasonEmptyTimeZone
	}
	if len(timeZoneName) > MaximumTimeZoneNameBytes {
		return ReasonTimeZoneTooLong
	}
	return ""
}
