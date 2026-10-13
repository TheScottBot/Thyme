// Command sizecheck fails the build when a release exe exceeds its budget.
// Run it on the release builds: go run ./tools/sizecheck thyme_amd64.exe thyme_arm64.exe
package main

import (
	"fmt"
	"os"

	"github.com/TheScottBot/Thyme/internal/repocheck"
)

// maximumReleaseExeBytes is the budget the author agreed in WE8 from WU6 as
// measured: 4,348,928 bytes on amd64 and 4,065,280 on arm64, stripped and
// trimmed, on 13 October 2026. 6 MiB leaves room for honest growth while
// still catching an accidental doubling.
const maximumReleaseExeBytes = 6 << 20

func main() {
	oversized, checkError := repocheck.FindOversizedFiles(os.Args[1:], maximumReleaseExeBytes)
	if checkError != nil {
		fmt.Fprintf(os.Stderr, "size check could not run: %v\n", checkError)
		os.Exit(2)
	}
	for _, file := range oversized {
		fmt.Printf("%s: %d bytes, over the %d byte budget\n", file.Path, file.SizeBytes, maximumReleaseExeBytes)
	}
	if len(oversized) > 0 {
		os.Exit(1)
	}
	fmt.Printf("sizecheck: %d file(s) within %d bytes\n", len(os.Args)-1, maximumReleaseExeBytes)
}
