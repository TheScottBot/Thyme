package repocheck

import "strings"

// networkImportPaths are the packages that would let Thyme talk to a network.
// Specification Part 6: there is no network, no listening socket and no
// telemetry, and none may be added without a new version of the
// specification. Each also forbids every path beneath it.
//
// The rule covers Thyme's own code only. golang.org/x/sys/windows imports
// net for socket type definitions, so net is linked into the exe all the
// same; that this never opens a socket was observed separately (security
// pass record, 13 October 2026).
var networkImportPaths = []string{"net", "crypto/tls", "golang.org/x/net"}

func isNetworkImport(importPath string) bool {
	for _, networkPath := range networkImportPaths {
		if importPath == networkPath || strings.HasPrefix(importPath, networkPath+"/") {
			return true
		}
	}
	return false
}

// FindNetworkImports applies the rule to every package, the shell and the
// entry point included, since the shell's exemption covers the operating
// system, not the network.
func FindNetworkImports(moduleRoot string) ([]ImportViolation, error) {
	isExempt := func(relativePath string) bool { return false }
	return findImports(moduleRoot, isExempt, isNetworkImport)
}
