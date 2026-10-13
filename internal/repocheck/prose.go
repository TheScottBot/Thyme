package repocheck

import (
	"bytes"
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"regexp"
	"strings"
	"unicode"
)

// ProseViolationReason is the category a scan hit is counted under.
type ProseViolationReason string

const (
	ReasonEmDash    ProseViolationReason = "em dash"
	ReasonEnDash    ProseViolationReason = "en dash"
	ReasonHyphenRun ProseViolationReason = "hyphen run"
)

const (
	emDashCharacter = '\u2014'
	enDashCharacter = '\u2013'
)

// ProseViolation names one forbidden dash or hyphen run and where it is.
type ProseViolation struct {
	FilePath   string
	LineNumber int
	Reason     ProseViolationReason
}

// markdownTableSeparatorRow matches a Markdown table separator row, made only
// of pipes, colons, spaces and hyphen runs. The datum exempts it, since
// Markdown needs the hyphens there and no converter produced them.
var markdownTableSeparatorRow = regexp.MustCompile(`^\s*\|?\s*:?-+:?\s*(\|\s*:?-+:?\s*)+\|?\s*$`)

// FindProseViolations scans every text file under scanRoot, except the exact
// vendored paths given, for em dashes, en dashes and converter mangled hyphen
// runs. Vendored paths are an explicit list, never a pattern, because a pattern
// eventually excludes something that is not vendored.
//
// Go files are tokenised and only comments and string literals are scanned,
// since a decrement operator is code rather than prose. Every other text file
// is scanned line by line. A file holding a NUL byte is treated as binary.
func FindProseViolations(scanRoot string, vendoredPaths []string) ([]ProseViolation, error) {
	var violations []ProseViolation

	skipDirectory := func(relativePath string, directoryName string) bool {
		return directoryName == ".git" || isVendoredPath(relativePath, vendoredPaths)
	}
	visitFile := func(absolutePath string, relativePath string) error {
		if isVendoredPath(relativePath, vendoredPaths) {
			return nil
		}
		fileContents, readError := os.ReadFile(absolutePath)
		if readError != nil {
			return fmt.Errorf("reading '%s': %w", relativePath, readError)
		}
		if bytes.IndexByte(fileContents, 0) >= 0 {
			return nil
		}
		var fileViolations []ProseViolation
		if strings.HasSuffix(relativePath, ".go") {
			goViolations, scanError := scanGoCommentsAndStrings(relativePath, fileContents)
			if scanError != nil {
				return scanError
			}
			fileViolations = goViolations
		} else {
			fileViolations = scanTextLines(relativePath, string(fileContents), 1)
		}
		violations = append(violations, fileViolations...)
		return nil
	}

	if walkError := walkFiles(scanRoot, skipDirectory, visitFile); walkError != nil {
		return nil, walkError
	}
	return violations, nil
}

func isVendoredPath(relativePath string, vendoredPaths []string) bool {
	for _, vendoredPath := range vendoredPaths {
		if relativePath == vendoredPath || strings.HasPrefix(relativePath, vendoredPath+"/") {
			return true
		}
	}
	return false
}

// scanGoCommentsAndStrings refuses a file the Go scanner cannot read, because
// a scan that silently skips a broken file would pass a file it never read.
func scanGoCommentsAndStrings(relativePath string, fileContents []byte) ([]ProseViolation, error) {
	var violations []ProseViolation
	var scanErrors scanner.ErrorList

	fileSet := token.NewFileSet()
	tokenFile := fileSet.AddFile(relativePath, fileSet.Base(), len(fileContents))
	var goScanner scanner.Scanner
	recordError := func(position token.Position, errorMessage string) {
		scanErrors.Add(position, errorMessage)
	}
	goScanner.Init(tokenFile, fileContents, recordError, scanner.ScanComments)

	for {
		tokenPosition, tokenKind, literalText := goScanner.Scan()
		if tokenKind == token.EOF {
			break
		}
		if tokenKind != token.COMMENT && tokenKind != token.STRING && tokenKind != token.CHAR {
			continue
		}
		startingLine := fileSet.Position(tokenPosition).Line
		violations = append(violations, scanTextLines(relativePath, literalText, startingLine)...)
	}

	if scanErrors.Len() > 0 {
		return nil, fmt.Errorf("scanning '%s': %w", relativePath, scanErrors.Err())
	}
	return violations, nil
}

func scanTextLines(relativePath string, text string, firstLineNumber int) []ProseViolation {
	var violations []ProseViolation
	for lineOffset, lineText := range strings.Split(text, "\n") {
		lineNumber := firstLineNumber + lineOffset
		for _, reason := range findLineViolations(lineText) {
			violations = append(violations, ProseViolation{FilePath: relativePath, LineNumber: lineNumber, Reason: reason})
		}
	}
	return violations
}

func findLineViolations(lineText string) []ProseViolationReason {
	var reasons []ProseViolationReason
	for _, character := range lineText {
		switch character {
		case emDashCharacter:
			reasons = append(reasons, ReasonEmDash)
		case enDashCharacter:
			reasons = append(reasons, ReasonEnDash)
		}
	}
	if markdownTableSeparatorRow.MatchString(lineText) {
		return reasons
	}

	lineRunes := []rune(lineText)
	for runStart := 0; runStart < len(lineRunes); {
		if lineRunes[runStart] != '-' {
			runStart++
			continue
		}
		runEnd := runStart
		for runEnd < len(lineRunes) && lineRunes[runEnd] == '-' {
			runEnd++
		}
		if runEnd-runStart >= 2 && !isCommandLineFlagToken(lineRunes, runStart, runEnd) {
			reasons = append(reasons, ReasonHyphenRun)
		}
		runStart = runEnd
	}
	return reasons
}

// isCommandLineFlagToken exempts a long option such as the race flag: exactly
// two hyphens, starting a token, immediately followed by a letter. A spaced
// pair of hyphens between words is the converter defect and is not exempt.
func isCommandLineFlagToken(lineRunes []rune, runStart int, runEnd int) bool {
	if runEnd-runStart != 2 || runEnd >= len(lineRunes) || !unicode.IsLetter(lineRunes[runEnd]) {
		return false
	}
	if runStart == 0 {
		return true
	}
	precedingCharacter := lineRunes[runStart-1]
	return unicode.IsSpace(precedingCharacter) || strings.ContainsRune("`'\"([", precedingCharacter)
}
