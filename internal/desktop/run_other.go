//go:build !windows && !(darwin && cgo)

package desktop

import (
	"fmt"
	"os"
)

// Run reports that this build has no shell. Thyme runs on Windows (WD10) and
// on macOS built with cgo (WD11, WD12). The pure packages build and test
// everywhere; this keeps the whole module building on the Linux runner too.
func Run() int {
	fmt.Fprintln(os.Stderr, "Thyme runs on Windows and macOS only; on macOS, build with cgo")
	return 1
}
