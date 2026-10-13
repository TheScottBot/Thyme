package repocheck

import (
	"fmt"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"
)

// shellPackageDirectories are the only directories allowed to touch the
// operating system. Everything else must build and test on any host with
// nothing but the Go toolchain, which is what keeps the rendering and timing
// decisions testable on the Linux runner. Exact directories, not prefixes, so
// a new package beside the shell does not inherit its exemption by name.
var shellPackageDirectories = []string{
	"internal/desktop",
	"cmd/thyme",
}

// forbiddenImportPaths lists imports that tie a package to an operating system
// or bypass memory safety. Each also forbids every path beneath it, so a new
// x/sys subpackage needs no change here.
var forbiddenImportPaths = []string{"syscall", "unsafe", "golang.org/x/sys"}

// ImportViolation names one forbidden import in a package outside the shell.
type ImportViolation struct {
	FilePath   string
	ImportPath string
}

func isForbiddenImport(importPath string) bool {
	for _, forbiddenPath := range forbiddenImportPaths {
		if importPath == forbiddenPath || strings.HasPrefix(importPath, forbiddenPath+"/") {
			return true
		}
	}
	return false
}

func isShellPackageDirectory(packageDirectory string) bool {
	for _, shellDirectory := range shellPackageDirectories {
		if packageDirectory == shellDirectory {
			return true
		}
	}
	return false
}

// FindForbiddenImports parses the import block of every Go file under
// moduleRoot, test files and files behind build constraints included, because
// a Windows only file in a pure package is exactly what the Linux build would
// never notice. A file that does not parse is an error rather than a pass.
func FindForbiddenImports(moduleRoot string) ([]ImportViolation, error) {
	isExempt := func(relativePath string) bool { return isShellPackageDirectory(path.Dir(relativePath)) }
	return findImports(moduleRoot, isExempt, isForbiddenImport)
}

// findImports is the walk both import rules share: every Go file the go tool
// would build, test files and files behind build constraints included,
// except those isExempt names, reporting each import isForbidden matches.
func findImports(moduleRoot string, isExempt func(relativePath string) bool, isForbidden func(importPath string) bool) ([]ImportViolation, error) {
	var violations []ImportViolation
	fileSet := token.NewFileSet()

	skipDirectory := func(relativePath string, directoryName string) bool {
		return isDirectoryTheGoToolIgnores(directoryName)
	}
	visitFile := func(absolutePath string, relativePath string) error {
		if !strings.HasSuffix(relativePath, ".go") || isExempt(relativePath) {
			return nil
		}
		parsedFile, parseError := parser.ParseFile(fileSet, absolutePath, nil, parser.ImportsOnly)
		if parseError != nil {
			return fmt.Errorf("parsing imports of '%s': %w", relativePath, parseError)
		}
		for _, importSpecification := range parsedFile.Imports {
			importPath, unquoteError := strconv.Unquote(importSpecification.Path.Value)
			if unquoteError != nil {
				return fmt.Errorf("reading an import path in '%s': %w", relativePath, unquoteError)
			}
			if isForbidden(importPath) {
				violations = append(violations, ImportViolation{FilePath: relativePath, ImportPath: importPath})
			}
		}
		return nil
	}

	if walkError := walkFiles(moduleRoot, skipDirectory, visitFile); walkError != nil {
		return nil, walkError
	}
	return violations, nil
}
