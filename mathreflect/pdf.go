package mathreflect

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ErrPDFtotextNotFound is returned when pdftotext is not in PATH.
var ErrPDFtotextNotFound = errors.New(
	"pdftotext not found in PATH; install with: brew install poppler (macOS) or apt install poppler-utils (Linux)",
)

var reProblemLine = regexp.MustCompile(`^([UJSOI])(\d+)\.\s+(.*)`)

// sectionCodes maps lowercase section keywords to section letter codes.
var sectionCodes = map[string]string{
	"junior":        "J",
	"senior":        "S",
	"olympiad":      "O",
	"undergraduate": "U",
	"individual":    "I",
}

// ParsedProblem is one problem extracted from PDF text.
type ParsedProblem struct {
	Section    string
	ProblemNum int
	Text       string
}

// ExtractTextFromPDF runs pdftotext on pdfBytes and returns plain text.
// Returns ErrPDFtotextNotFound if pdftotext is not in PATH.
func ExtractTextFromPDF(pdfBytes []byte) (string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", ErrPDFtotextNotFound
	}

	f, err := os.CreateTemp("", "mr_*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(pdfBytes); err != nil {
		f.Close()
		return "", fmt.Errorf("write pdf: %w", err)
	}
	f.Close()

	out, err := exec.Command("pdftotext", "-layout", f.Name(), "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	return string(out), nil
}

// ParseProblemsFromText parses pdftotext output into individual problems.
func ParseProblemsFromText(text string) []ParsedProblem {
	lines := strings.Split(text, "\n")
	var problems []ParsedProblem

	currentSection := "O"
	var currentProb *ParsedProblem
	var buf strings.Builder

	flush := func() {
		if currentProb != nil {
			currentProb.Text = strings.TrimSpace(buf.String())
			if currentProb.Text != "" {
				problems = append(problems, *currentProb)
			}
			currentProb = nil
			buf.Reset()
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip page headers like "Mathematical Reflections 3 (2026)".
		if strings.HasPrefix(trimmed, "Mathematical Reflections") {
			continue
		}

		// Detect section headers.
		lower := strings.ToLower(trimmed)
		for keyword, code := range sectionCodes {
			if strings.Contains(lower, keyword) && strings.Contains(lower, "problem") {
				flush()
				currentSection = code
				break
			}
		}

		// Detect problem start: "J715.  Find all..." or "O12.  Let ABC..."
		m := reProblemLine.FindStringSubmatch(trimmed)
		if m != nil {
			flush()
			num, _ := strconv.Atoi(m[2])
			currentSection = m[1]
			currentProb = &ParsedProblem{
				Section:    currentSection,
				ProblemNum: num,
			}
			buf.WriteString(m[3])
			buf.WriteString("\n")
			continue
		}

		if currentProb != nil {
			buf.WriteString(line)
			buf.WriteString("\n")
		}
	}
	flush()

	return problems
}
