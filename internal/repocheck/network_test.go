package repocheck

import "testing"

func TestFindNetworkImports_FlagsEveryNetworkImportInEveryPackage(t *testing.T) {
	testCases := []struct {
		name         string
		relativePath string
		importPath   string
	}{
		{name: "net in the shell, which may use syscall but not the network", relativePath: "internal/desktop/update.go", importPath: "net"},
		{name: "net/http in the entry point", relativePath: "cmd/thyme/fetch.go", importPath: "net/http"},
		{name: "net/netip in a pure package", relativePath: "internal/render/addresses.go", importPath: "net/netip"},
		{name: "crypto/tls in a tool", relativePath: "tools/repocheck/upload.go", importPath: "crypto/tls"},
		{name: "x/net in a test file", relativePath: "internal/settings/remote_test.go", importPath: "golang.org/x/net/html"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fixtureRoot := writeFixtureTree(t, map[string]string{
				testCase.relativePath: "package planted\n\nimport _ \"" + testCase.importPath + "\"\n",
			})

			violations, findError := FindNetworkImports(fixtureRoot)
			if findError != nil {
				t.Fatalf("unexpected error: %v", findError)
			}

			if len(violations) != 1 || violations[0].ImportPath != testCase.importPath || violations[0].FilePath != testCase.relativePath {
				t.Errorf("expected %s in %s flagged, got %+v", testCase.importPath, testCase.relativePath, violations)
			}
		})
	}
}

func TestFindNetworkImports_DoesNotFlagPathsThatOnlyStartWithNet(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/render/render.go": "package render\n\nimport (\n\t_ \"netstuff/example\"\n\t_ \"crypto/sha256\"\n\t_ \"golang.org/x/network\"\n)\n",
	})

	violations, findError := FindNetworkImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Errorf("expected only real network packages to be flagged, got %+v", violations)
	}
}

func TestFindNetworkImports_IgnoresDirectoriesTheGoToolIgnores(t *testing.T) {
	fixtureRoot := writeFixtureTree(t, map[string]string{
		"internal/render/testdata/sample.go": "package sample\n\nimport _ \"net\"\n",
	})

	violations, findError := FindNetworkImports(fixtureRoot)
	if findError != nil {
		t.Fatalf("unexpected error: %v", findError)
	}

	if len(violations) != 0 {
		t.Errorf("expected testdata to be skipped, got %+v", violations)
	}
}
