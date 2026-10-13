package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// The settings file's member names. The parser and the encoder both read them
// from fieldCodecs below, so the grammar exists once (datum 0.7).
const (
	fieldNameVersion      = "version"
	fieldNamePosition     = "position"
	fieldNameScalePercent = "scalePercent"
	fieldNameAlwaysOnTop  = "alwaysOnTop"
	fieldNameShowInTray   = "showInTray"
	fieldNameShowSeconds  = "showSeconds"
	fieldNameZones        = "zones"

	memberNameX        = "x"
	memberNameY        = "y"
	memberNameLabel    = "label"
	memberNameTimeZone = "timeZone"
)

// fieldCodec is one top level member of the file. decode assigns to target
// only on success, so a rejected field keeps the default it started with.
type fieldCodec struct {
	name   string
	decode func(rawValue json.RawMessage, target *Settings) RejectionReason
	encode func(source Settings) any
}

// fieldCodecs is the settings grammar, in the order the file is written.
var fieldCodecs = []fieldCodec{
	{name: fieldNamePosition, decode: decodePosition, encode: encodePosition},
	{name: fieldNameScalePercent, decode: decodeScalePercent, encode: func(source Settings) any { return source.ScalePercent }},
	{name: fieldNameAlwaysOnTop, decode: decodeBooleanInto(func(target *Settings) *bool { return &target.AlwaysOnTop }), encode: func(source Settings) any { return source.AlwaysOnTop }},
	{name: fieldNameShowInTray, decode: decodeBooleanInto(func(target *Settings) *bool { return &target.ShowInTray }), encode: func(source Settings) any { return source.ShowInTray }},
	{name: fieldNameShowSeconds, decode: decodeBooleanInto(func(target *Settings) *bool { return &target.ShowSeconds }), encode: func(source Settings) any { return source.ShowSeconds }},
	{name: fieldNameZones, decode: decodeZones, encode: encodeZones},
}

// Source says where the settings in use came from, for the start up log.
type Source string

const (
	SourceFile     Source = "file"
	SourceDefaults Source = "defaults"
)

// LoadReport says what happened to a settings file. OverwriteAllowed is false
// when the file existed but was refused whole: saving over it would destroy a
// file Thyme could not understand, which may be the user's broken hand edit or
// a newer version's settings.
type LoadReport struct {
	Source           Source
	Rejections       []Rejection
	OverwriteAllowed bool
}

func refusedWhole(field string, reason RejectionReason) (Settings, LoadReport) {
	return Defaults(), LoadReport{
		Source:           SourceDefaults,
		Rejections:       []Rejection{{Field: field, Reason: reason}},
		OverwriteAllowed: false,
	}
}

// Parse reads a settings document. Problems with the whole document (too
// large, not one JSON object, no or another version) yield the defaults. A
// problem with one field yields that field's default, and every rejection is
// counted by reason. Parse never fails and never returns invalid settings.
func Parse(document []byte) (Settings, LoadReport) {
	if len(document) > MaximumFileBytes {
		return refusedWhole("", ReasonFileTooLarge)
	}
	members, malformed := decodeObjectMembers(document)
	if malformed {
		return refusedWhole("", ReasonMalformedDocument)
	}

	versionMember, hasVersion := members.byName[fieldNameVersion]
	if !hasVersion {
		return refusedWhole(fieldNameVersion, ReasonVersionMissing)
	}
	if members.duplicated[fieldNameVersion] {
		return refusedWhole(fieldNameVersion, ReasonVersionUnsupported)
	}
	if version, reason := decodeInteger(versionMember, 32); reason != "" || version != supportedVersion {
		return refusedWhole(fieldNameVersion, ReasonVersionUnsupported)
	}

	parsed := Defaults()
	report := LoadReport{Source: SourceFile, OverwriteAllowed: true}
	reject := func(field string, reason RejectionReason) {
		report.Rejections = append(report.Rejections, Rejection{Field: field, Reason: reason})
	}

	for _, memberName := range members.namesInOrder {
		if memberName != fieldNameVersion && !isKnownFieldName(memberName) {
			reject("", ReasonUnknownField)
		}
	}
	for _, codec := range fieldCodecs {
		rawValue, present := members.byName[codec.name]
		switch {
		case members.duplicated[codec.name]:
			reject(codec.name, ReasonDuplicateField)
		case !present:
			reject(codec.name, ReasonMissingField)
		default:
			if reason := codec.decode(rawValue, &parsed); reason != "" {
				reject(codec.name, reason)
			}
		}
	}
	return parsed, report
}

func isKnownFieldName(memberName string) bool {
	for _, codec := range fieldCodecs {
		if codec.name == memberName {
			return true
		}
	}
	return false
}

// Encode writes settings in the file's grammar, refusing any that would not
// load back, so a save can never write a file the next start rejects.
func Encode(settings Settings) ([]byte, error) {
	if rejections := Validate(settings); len(rejections) > 0 {
		return nil, fmt.Errorf("refusing to encode invalid settings: %v", rejections)
	}

	var compactDocument bytes.Buffer
	compactDocument.WriteString(`{"` + fieldNameVersion + `":` + strconv.Itoa(supportedVersion))
	for _, codec := range fieldCodecs {
		encodedValue, marshalError := json.Marshal(codec.encode(settings))
		if marshalError != nil {
			return nil, fmt.Errorf("encoding '%s': %w", codec.name, marshalError)
		}
		compactDocument.WriteString(`,"` + codec.name + `":`)
		compactDocument.Write(encodedValue)
	}
	compactDocument.WriteString("}")

	var indentedDocument bytes.Buffer
	if indentError := json.Indent(&indentedDocument, compactDocument.Bytes(), "", "  "); indentError != nil {
		return nil, fmt.Errorf("indenting settings: %w", indentError)
	}
	indentedDocument.WriteString("\n")
	return indentedDocument.Bytes(), nil
}

// objectMembers is one JSON object read member by member, keeping the order
// and noting duplicates, which encoding/json would otherwise resolve silently
// by letting the last one win.
type objectMembers struct {
	namesInOrder []string
	byName       map[string]json.RawMessage
	duplicated   map[string]bool
}

// decodeObjectMembers reads exactly one JSON object and nothing after it.
// Anything else (another type, a syntax error, a second value) is malformed.
func decodeObjectMembers(document []byte) (objectMembers, bool) {
	members := objectMembers{byName: map[string]json.RawMessage{}, duplicated: map[string]bool{}}
	decoder := json.NewDecoder(bytes.NewReader(document))

	openingToken, tokenError := decoder.Token()
	if tokenError != nil || openingToken != json.Delim('{') {
		return objectMembers{}, true
	}
	for decoder.More() {
		nameToken, nameError := decoder.Token()
		memberName, isString := nameToken.(string)
		if nameError != nil || !isString {
			return objectMembers{}, true
		}
		var rawValue json.RawMessage
		if valueError := decoder.Decode(&rawValue); valueError != nil {
			return objectMembers{}, true
		}
		if _, seen := members.byName[memberName]; seen {
			members.duplicated[memberName] = true
			continue
		}
		members.namesInOrder = append(members.namesInOrder, memberName)
		members.byName[memberName] = rawValue
	}
	if closingToken, closingError := decoder.Token(); closingError != nil || closingToken != json.Delim('}') {
		return objectMembers{}, true
	}
	if _, trailingError := decoder.Token(); !errors.Is(trailingError, io.EOF) {
		return objectMembers{}, true
	}
	return members, false
}

// decodeInteger accepts only a plain JSON integer. A quoted number, a
// fraction or an exponent is the wrong type; a number too big for bitSize is
// out of range.
func decodeInteger(rawValue json.RawMessage, bitSize int) (int64, RejectionReason) {
	text := string(bytes.TrimSpace(rawValue))
	if text == "" || (text[0] != '-' && (text[0] < '0' || text[0] > '9')) {
		return 0, ReasonWrongType
	}
	value, parseError := strconv.ParseInt(text, 10, bitSize)
	if errors.Is(parseError, strconv.ErrRange) {
		return 0, ReasonOutOfRange
	}
	if parseError != nil {
		return 0, ReasonWrongType
	}
	return value, ""
}

// decodeNonNull decodes into a pointer so that a JSON null, which
// encoding/json would quietly treat as the zero value, is the wrong type.
func decodeNonNull[ValueType any](rawValue json.RawMessage) (ValueType, RejectionReason) {
	var decoded *ValueType
	if unmarshalError := json.Unmarshal(rawValue, &decoded); unmarshalError != nil || decoded == nil {
		var zeroValue ValueType
		return zeroValue, ReasonWrongType
	}
	return *decoded, ""
}

func decodeBooleanInto(fieldOf func(target *Settings) *bool) func(json.RawMessage, *Settings) RejectionReason {
	return func(rawValue json.RawMessage, target *Settings) RejectionReason {
		value, reason := decodeNonNull[bool](rawValue)
		if reason != "" {
			return reason
		}
		*fieldOf(target) = value
		return ""
	}
}

func decodeScalePercent(rawValue json.RawMessage, target *Settings) RejectionReason {
	scalePercent, reason := decodeInteger(rawValue, 32)
	if reason != "" {
		return reason
	}
	if reason := validateScalePercent(int(scalePercent)); reason != "" {
		return reason
	}
	target.ScalePercent = int(scalePercent)
	return ""
}

// decodeExactMembers reads a nested object that must hold exactly the named
// members, each once.
func decodeExactMembers(rawValue json.RawMessage, requiredNames ...string) (map[string]json.RawMessage, RejectionReason) {
	members, malformed := decodeObjectMembers(rawValue)
	if malformed {
		return nil, ReasonWrongType
	}
	if len(members.duplicated) > 0 {
		return nil, ReasonDuplicateField
	}
	if len(members.byName) > len(requiredNames) {
		return nil, ReasonUnknownField
	}
	for _, requiredName := range requiredNames {
		if _, present := members.byName[requiredName]; !present {
			if len(members.byName) == len(requiredNames) {
				return nil, ReasonUnknownField
			}
			return nil, ReasonMissingField
		}
	}
	return members.byName, ""
}

func decodePosition(rawValue json.RawMessage, target *Settings) RejectionReason {
	members, reason := decodeExactMembers(rawValue, memberNameX, memberNameY)
	if reason != "" {
		return reason
	}
	horizontal, reason := decodeInteger(members[memberNameX], 32)
	if reason != "" {
		return reason
	}
	vertical, reason := decodeInteger(members[memberNameY], 32)
	if reason != "" {
		return reason
	}
	target.Position = Position{X: int32(horizontal), Y: int32(vertical)}
	return ""
}

func encodePosition(source Settings) any {
	return map[string]int32{memberNameX: source.Position.X, memberNameY: source.Position.Y}
}

// decodeZones counts entries before decoding any of them, so a long list is
// refused without the work of reading it.
func decodeZones(rawValue json.RawMessage, target *Settings) RejectionReason {
	entries, reason := decodeNonNull[[]json.RawMessage](rawValue)
	if reason != "" {
		return reason
	}
	if len(entries) > MaximumZoneCount {
		return ReasonTooManyZones
	}
	zones := make([]ZoneSetting, 0, len(entries))
	for _, entry := range entries {
		members, reason := decodeExactMembers(entry, memberNameLabel, memberNameTimeZone)
		if reason != "" {
			return reason
		}
		label, reason := decodeNonNull[string](members[memberNameLabel])
		if reason != "" {
			return reason
		}
		timeZone, reason := decodeNonNull[string](members[memberNameTimeZone])
		if reason != "" {
			return reason
		}
		zones = append(zones, ZoneSetting{Label: label, TimeZone: timeZone})
	}
	if reason := validateZones(zones); reason != "" {
		return reason
	}
	target.Zones = zones
	return ""
}

func encodeZones(source Settings) any {
	entries := make([]map[string]string, 0, len(source.Zones))
	for _, zone := range source.Zones {
		entries = append(entries, map[string]string{memberNameLabel: zone.Label, memberNameTimeZone: zone.TimeZone})
	}
	return entries
}
