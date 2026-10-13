package eventlog

import (
	"errors"
	"io/fs"
	"os"
	"strings"
)

// ReasonWithoutPath is an error's text fit for the log: a file error's
// operation and the operating system's reason, with the paths removed, since
// a path under the user's profile carries their name (specification Part 5).
// Any wrapping context around the file error is kept.
func ReasonWithoutPath(failure error) string {
	fullText := failure.Error()
	var pathError *fs.PathError
	if errors.As(failure, &pathError) {
		return strings.Replace(fullText, pathError.Error(), pathError.Op+": "+pathError.Err.Error(), 1)
	}
	var linkError *os.LinkError
	if errors.As(failure, &linkError) {
		return strings.Replace(fullText, linkError.Error(), linkError.Op+": "+linkError.Err.Error(), 1)
	}
	return fullText
}
