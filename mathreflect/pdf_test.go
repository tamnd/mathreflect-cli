package mathreflect_test

import (
	"testing"

	"github.com/tamnd/mathreflect-cli/mathreflect"
)

const samplePDFText = `
Mathematical Reflections 3 (2026)

Junior Problems

J715.  Find all integers n such that n^2 + 1 is divisible by 5.

J716.  Let a, b, c be positive integers with a + b + c = 10.
       Prove that a^2 + b^2 + c^2 >= 34.

Olympiad Problems

O12.  Let ABC be a triangle with circumradius R.
      Prove that the sum of the squares of the sides
      equals 9R^2 - R^2.

Mathematical Reflections 3 (2026)

Senior Problems

S45.  Find all polynomials P(x) with integer coefficients
      such that P(n) divides n! for all positive integers n.
`

func TestParseProblemsFromText(t *testing.T) {
	problems := mathreflect.ParseProblemsFromText(samplePDFText)
	if len(problems) == 0 {
		t.Fatal("expected at least one problem, got 0")
	}

	// Check first problem
	found := false
	for _, p := range problems {
		if p.Section == "J" && p.ProblemNum == 715 {
			found = true
			if p.Text == "" {
				t.Errorf("J715 has empty text")
			}
		}
	}
	if !found {
		t.Errorf("problem J715 not found; got: %+v", problems)
	}

	// Check multi-line problem
	for _, p := range problems {
		if p.Section == "J" && p.ProblemNum == 716 {
			if p.Text == "" {
				t.Errorf("J716 has empty text")
			}
		}
		if p.Section == "O" && p.ProblemNum == 12 {
			if p.Text == "" {
				t.Errorf("O12 has empty text")
			}
		}
	}

	// Page headers should NOT appear in problem text.
	for _, p := range problems {
		if len(p.Text) > 0 && p.Text[:min(len(p.Text), 26)] == "Mathematical Reflections 3" {
			t.Errorf("page header leaked into problem text for %s%d", p.Section, p.ProblemNum)
		}
	}
}

func TestProblemID(t *testing.T) {
	id := mathreflect.ProblemID(2026, 3, "O", 5)
	if id != "2026-3-O-5" {
		t.Errorf("ProblemID = %q, want %q", id, "2026-3-O-5")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
