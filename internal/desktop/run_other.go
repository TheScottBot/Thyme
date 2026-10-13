//go:build !windows

package desktop

import (
	"fmt"
	"os"
)

// Run reports that the shell exists only for Windows (WD10). The pure
// packages build and test everywhere; this keeps the whole module building
// on the Linux runner too.
func Run() int {
	fmt.Fprintln(os.Stderr, "Thyme runs on Windows only")
	return 1
}
