// Command thyme is the entry point. It only wires the packages together;
// every behaviour lives in internal/.
package main

import (
	"os"

	"github.com/TheScottBot/Thyme/internal/desktop"
)

func main() {
	os.Exit(desktop.Run())
}
