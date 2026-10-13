package settings

import (
	"reflect"
	"testing"
)

// FuzzParse holds three properties for any input: Parse never panics, what it
// returns always passes Validate, and a document it accepts cleanly survives
// an encode and parse unchanged.
func FuzzParse(f *testing.F) {
	encodedDefaults, encodeError := Encode(Defaults())
	if encodeError != nil {
		f.Fatalf("encoding defaults for the seed corpus: %v", encodeError)
	}
	encodedChanged, encodeError := Encode(everyFieldChanged())
	if encodeError != nil {
		f.Fatalf("encoding changed settings for the seed corpus: %v", encodeError)
	}
	f.Add(encodedDefaults)
	f.Add(encodedChanged)
	f.Add([]byte(`{"version":1,"zones":[{"label":"SÃO","timeZone":"America/Sao_Paulo"}]}`))
	f.Add([]byte(`{"version":2}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, document []byte) {
		parsed, report := Parse(document)

		if rejections := Validate(parsed); len(rejections) != 0 {
			t.Fatalf("Parse returned invalid settings %+v: %+v", parsed, rejections)
		}
		if len(report.Rejections) != 0 {
			return
		}
		reencoded, encodeError := Encode(parsed)
		if encodeError != nil {
			t.Fatalf("cleanly parsed settings would not encode: %v", encodeError)
		}
		reparsed, reparseReport := Parse(reencoded)
		if !reflect.DeepEqual(reparsed, parsed) || len(reparseReport.Rejections) != 0 {
			t.Fatalf("round trip changed %+v into %+v with %+v", parsed, reparsed, reparseReport.Rejections)
		}
	})
}
