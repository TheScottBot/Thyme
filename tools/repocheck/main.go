// Command repocheck runs the tree wide checks over the repository it is started
// in and exits non zero on any hit, which is what fails the build in continuous
// integration. Run it from the repository root: go run ./tools/repocheck
package main

import (
	"fmt"
	"os"

	"github.com/TheScottBot/Thyme/internal/repocheck"
)

// vendoredPaths is the explicit list of third party files kept verbatim under
// datum 0.13 and therefore exempt from the prose scan. Each entry is an exact
// file or directory, never a pattern. It is empty until something is vendored.
var vendoredPaths = []string{}

func main() {
	os.Exit(checkRepository("."))
}

func checkRepository(repositoryRoot string) int {
	failureCount := 0

	importViolations, importError := repocheck.FindForbiddenImports(repositoryRoot)
	if importError != nil {
		fmt.Fprintf(os.Stderr, "import boundary check could not run: %v\n", importError)
		return 2
	}
	for _, violation := range importViolations {
		fmt.Printf("%s: forbidden import '%s' outside the Windows shell\n", violation.FilePath, violation.ImportPath)
		failureCount++
	}

	proseViolations, proseError := repocheck.FindProseViolations(repositoryRoot, vendoredPaths)
	if proseError != nil {
		fmt.Fprintf(os.Stderr, "prose scan could not run: %v\n", proseError)
		return 2
	}
	for _, violation := range proseViolations {
		fmt.Printf("%s:%d: %s\n", violation.FilePath, violation.LineNumber, violation.Reason)
		failureCount++
	}

	if failureCount > 0 {
		fmt.Printf("repocheck: %d violation(s)\n", failureCount)
		return 1
	}
	fmt.Println("repocheck: clean")
	return 0
}
