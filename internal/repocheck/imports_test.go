package repocheck

import (
	"testing"
)

func TestFindForbiddenImports_FlagsXSysWindowsPlantedInRender(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/render/render.go": "package render\n\nimport _ \"golang.org/x/sys/windows\"\n",
	})

	violations, findError := FindForbiddenImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 1 {
		t.Fatalf("expected exactly one violation, got %d: %+v", len(violations), violations)
	}
	if violations[0].FilePath != "internal/render/render.go" {
		t.Errorf("expected the planted file to be named, got '%s'", violations[0].FilePath)
	}
	if violations[0].ImportPath != "golang.org/x/sys/windows" {
		t.Errorf("expected the planted import to be named, got '%s'", violations[0].ImportPath)
	}
}

func TestFindForbiddenImports_FlagsEveryForbiddenImportInPurePackages(t *testing.T) {
	testCases := []struct {
		name       string
		importPath string
	}{
		{name: "syscall is operating system specific", importPath: "syscall"},
		{name: "unsafe is confined to the shell", importPath: "unsafe"},
		{name: "the x/sys root is forbidden", importPath: "golang.org/x/sys"},
		{name: "any x/sys subpackage is forbidden", importPath: "golang.org/x/sys/unix"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixtureRoot := writeFixtureTree(t, map[string]string{
				"internal/clockmodel/clock.go": "package clockmodel\n\nimport _ \"" + testCase.importPath + "\"\n",
			})

			violations, findError := FindForbiddenImports(fixtureRoot)
			if findError != nil {
				t.Fatalf("unexpected error: %v", findError)
			}

			if len(violations) != 1 || violations[0].ImportPath != testCase.importPath {
				t.Fatalf("expected one violation for '%s', got %+v", testCase.importPath, violations)
			}
		})
	}
}

func TestFindForbiddenImports_FlagsForbiddenImportInTestFile(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/settings/settings_test.go": "package settings\n\nimport _ \"unsafe\"\n",
	})

	violations, findError := FindForbiddenImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 1 {
		t.Fatalf("expected the test file's import to be flagged, got %+v", violations)
	}
}

func TestFindForbiddenImports_FlagsForbiddenImportBehindBuildConstraint(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/placement/placement_windows.go": "//go:build windows\n\npackage placement\n\nimport _ \"syscall\"\n",
	})

	violations, findError := FindForbiddenImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 1 {
		t.Fatalf("expected a Windows only file to be checked on every host, got %+v", violations)
	}
}

func TestFindForbiddenImports_AllowsShellAndEntryPoint(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/desktop/window.go": "package desktop\n\nimport (\n\t_ \"syscall\"\n\t_ \"unsafe\"\n\t_ \"golang.org/x/sys/windows\"\n)\n",
		"cmd/thyme/main.go":          "package main\n\nimport _ \"golang.org/x/sys/windows\"\n\nfunc main() {}\n",
	})

	violations, findError := FindForbiddenImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Fatalf("expected the shell and entry point to be allowed, got %+v", violations)
	}
}

func TestFindForbiddenImports_DoesNotAllowPackageNestedUnderShellName(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/desktopextras/extras.go": "package desktopextras\n\nimport _ \"unsafe\"\n",
	})

	violations, findError := FindForbiddenImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 1 {
		t.Fatalf("expected a name sharing the shell's prefix to stay forbidden, got %+v", violations)
	}
}

func TestFindForbiddenImports_IgnoresDirectoriesTheGoToolIgnores(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/render/testdata/sample.go": "package sample\n\nimport _ \"unsafe\"\n",
		".git/hooks/sample.go":               "package sample\n\nimport _ \"unsafe\"\n",
		"_scratch/sample.go":                 "package sample\n\nimport _ \"unsafe\"\n",
	})

	violations, findError := FindForbiddenImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Fatalf("expected testdata, dot and underscore directories to be skipped, got %+v", violations)
	}
}

func TestFindForbiddenImports_ReportsNothingForCleanTree(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/theme/theme.go": "package theme\n\nimport \"image/color\"\n\nvar PanelFill = color.RGBA{}\n",
	})

	violations, findError := FindForbiddenImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
}

func TestFindForbiddenImports_ReturnsErrorForUnparseableFile(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/render/broken.go": "package render\n\nimport (\n",
	})

	if _, findError := FindForbiddenImports(fixtureRoot); findError == nil {
		t.Fatal("expected an unparseable file to be an error rather than silently skipped")
	}
}
