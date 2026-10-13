package repocheck

import (
	"strings"
	"testing"
)

// The forbidden characters are built from escapes and repetition so that this
// file itself passes the scan it tests.
var (
	emDash       = "\u2014"
	enDash       = "\u2013"
	doubleHyphen = strings.Repeat("-", 2)
	tripleHyphen = strings.Repeat("-", 3)
)

func TestFindProseViolations_FlagsEachPlantedRunInGoComment(t *testing.T) {
	testCases := []struct {
		name           string
		planted        string
		expectedReason ProseViolationReason
	}{
		{name: "em dash", planted: emDash, expectedReason: ReasonEmDash},
		{name: "en dash", planted: enDash, expectedReason: ReasonEnDash},
		{name: "double hyphen run", planted: doubleHyphen, expectedReason: ReasonHyphenRun},
		{name: "triple hyphen run", planted: tripleHyphen, expectedReason: ReasonHyphenRun},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixtureRoot := writeFixtureTree(t, map[string]string{
				"internal/clockmodel/clock.go": "package clockmodel\n\n// The wake is late " + testCase.planted + " by design.\nvar wakeMargin = 20\n",
			})

			violations, findError := FindProseViolations(fixtureRoot, nil)
			if findError != nil {
				t.Fatalf("unexpected error: %v", findError)
			}

			if len(violations) != 1 {
				t.Fatalf("expected one violation, got %+v", violations)
			}
			if violations[0].Reason != testCase.expectedReason {
				t.Errorf("expected reason '%s', got '%s'", testCase.expectedReason, violations[0].Reason)
			}
			if violations[0].FilePath != "internal/clockmodel/clock.go" || violations[0].LineNumber != 3 {
				t.Errorf("expected internal/clockmodel/clock.go line 3, got %s line %d", violations[0].FilePath, violations[0].LineNumber)
			}
		})
	}
}

func TestFindProseViolations_FlagsRunInGoStringLiteral(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/desktop/menu.go": "package desktop\n\nconst exitLabel = \"Exit " + emDash + " now\"\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, nil)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 1 || violations[0].Reason != ReasonEmDash {
		t.Fatalf("expected the user facing string to be flagged, got %+v", violations)
	}
}

func TestFindProseViolations_IgnoresGoDecrementOperator(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/render/blur.go": "package render\n\nfunc countDown(remaining int) int {\n\tremaining" + doubleHyphen + "\n\treturn remaining\n}\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, nil)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Fatalf("expected Go code outside comments and strings to be exempt, got %+v", violations)
	}
}

func TestFindProseViolations_FlagsEachRunInMarkdownAndYaml(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"README.md":                "# Thyme\n\nA clock " + enDash + " small.\n",
		".github/workflows/ci.yml": "name: ci\n# checks " + doubleHyphen + " all of them\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, nil)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 2 {
		t.Fatalf("expected one violation per file, got %+v", violations)
	}
}

func TestFindProseViolations_ExemptsCommandLineFlagTokens(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"TESTING.md": "Run `go test " + doubleHyphen + "race ./...` or go test " + doubleHyphen + "count=1.\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, nil)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Fatalf("expected flag tokens to be exempt, got %+v", violations)
	}
}

func TestFindProseViolations_StillFlagsSpacedRunBesideFlagToken(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"TESTING.md": "Use " + doubleHyphen + "race " + doubleHyphen + " it is cheap.\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, nil)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 1 {
		t.Fatalf("expected the spaced run to be flagged and the flag token exempt, got %+v", violations)
	}
}

func TestFindProseViolations_ExemptsMarkdownTableSeparatorRow(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"TESTING.md": "| Suite | Command |\n|" + tripleHyphen + "|:" + tripleHyphen + ":|\n| a | b |\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, nil)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Fatalf("expected the table separator row to be exempt, got %+v", violations)
	}
}

func TestFindProseViolations_ExcludesVendoredPathsByExactListOnly(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"third_party/font/OFL.txt":         "Copyright " + emDash + " upstream.\n",
		"third_party/fontnotes/README.md":  "Ours " + emDash + " not vendored.\n",
		"third_party/font/OFL.txt.ours.md": "Ours " + emDash + " not vendored.\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, []string{"third_party/font/OFL.txt"})
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 2 {
		t.Fatalf("expected only the exact vendored file to be excluded, got %+v", violations)
	}
	for _, violation := range violations {
		if violation.FilePath == "third_party/font/OFL.txt" {
			t.Errorf("vendored file was scanned: %+v", violation)
		}
	}
}

func TestFindProseViolations_ExcludesVendoredDirectoryAndItsContents(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"third_party/font/OFL.txt":     "Copyright " + emDash + " upstream.\n",
		"third_party/font/sub/note.md": "Upstream " + emDash + " too.\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, []string{"third_party/font"})
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Fatalf("expected the vendored directory to be excluded, got %+v", violations)
	}
}

func TestFindProseViolations_SkipsBinaryFilesAndGitDirectory(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"docs/theme/mockup.png": "\x89PNG\x00\x00" + emDash,
		".git/COMMIT_EDITMSG":   "message " + emDash + "\n",
	})

	violations, findError := FindProseViolations(fixtureRoot, nil)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Fatalf("expected binary files and the git directory to be skipped, got %+v", violations)
	}
}

func TestFindProseViolations_ReturnsErrorForUnparseableGoFile(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/render/broken.go": "package render\n\nconst unterminated = \"\n",
	})

	if _, findError := FindProseViolations(fixtureRoot, nil); findError == nil {
		t.Fatal("expected an unscannable Go file to be an error rather than silently passed")
	}
}
