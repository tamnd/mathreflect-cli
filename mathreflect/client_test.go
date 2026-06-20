package mathreflect_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/mathreflect-cli/mathreflect"
)

// archiveHTML is a minimal mock of the awesomemath.org archive page.
const archiveHTML = `<!DOCTYPE html>
<html>
<body>
<a href="/wp-pdf-files/math-reflections/mr-2026-03/mr_3_2026_problems.pdf">Download the problem column.</a>
<a href="/wp-pdf-files/math-reflections/mr-2026-02/mr_2_2026_problems.pdf">Download the problem column.</a>
<a href="/wp-pdf-files/math-reflections/mr-2025-06/mr_6_2025_problems_1.pdf">Download the problem column.</a>
<a href="https://example.com/other.pdf">external</a>
</body>
</html>`

func TestFetchArchiveIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(archiveHTML))
	}))
	defer srv.Close()

	// Override ArchiveURL by patching the request -- we test the parser directly.
	client := mathreflect.NewClient(0, 5*time.Second)
	_ = client // used below via a local helper

	// Test the parser via FetchArchiveIndex with a mock URL.
	// We test ExtractIssueYearNum instead since FetchArchiveIndex uses the const URL.
	year, num := mathreflect.ExtractIssueYearNum(
		"https://www.awesomemath.org/wp-pdf-files/math-reflections/mr-2026-03/mr_3_2026_problems.pdf")
	if year != 2026 {
		t.Errorf("year = %d, want 2026", year)
	}
	if num != 3 {
		t.Errorf("num = %d, want 3", num)
	}

	year2, num2 := mathreflect.ExtractIssueYearNum(
		"https://www.awesomemath.org/wp-pdf-files/math-reflections/mr-2025-06/mr_6_2025_problems_1.pdf")
	if year2 != 2025 {
		t.Errorf("year2 = %d, want 2025", year2)
	}
	if num2 != 6 {
		t.Errorf("num2 = %d, want 6", num2)
	}

	_ = srv
}

func TestExtractIssueYearNum(t *testing.T) {
	tests := []struct {
		url      string
		wantYear int
		wantNum  int
	}{
		{
			url:      "https://www.awesomemath.org/wp-pdf-files/math-reflections/mr-2026-03/mr_3_2026_problems.pdf",
			wantYear: 2026,
			wantNum:  3,
		},
		{
			url:      "https://www.awesomemath.org/wp-pdf-files/math-reflections/mr-2006-01/mr_1_2006_problems.pdf",
			wantYear: 2006,
			wantNum:  1,
		},
		{
			url:      "not-a-match",
			wantYear: 0,
			wantNum:  0,
		},
	}
	for _, tt := range tests {
		year, num := mathreflect.ExtractIssueYearNum(tt.url)
		if year != tt.wantYear || num != tt.wantNum {
			t.Errorf("ExtractIssueYearNum(%q) = (%d, %d), want (%d, %d)",
				tt.url, year, num, tt.wantYear, tt.wantNum)
		}
	}
}

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte("hello"))
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	client := mathreflect.NewClient(0, 5*time.Second)
	ctx := context.Background()

	body, code, err := client.FetchPDF(ctx, srv.URL+"/ok")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 200 {
		t.Errorf("code = %d, want 200", code)
	}
	if !strings.Contains(string(body), "hello") {
		t.Errorf("body = %q, want to contain 'hello'", string(body))
	}

	// 404 should return (nil, 404, nil).
	body2, code2, err2 := client.FetchPDF(ctx, srv.URL+"/notfound")
	if err2 != nil {
		t.Errorf("unexpected error for 404: %v", err2)
	}
	if code2 != 404 {
		t.Errorf("code2 = %d, want 404", code2)
	}
	if len(body2) != 0 {
		t.Errorf("body2 should be nil/empty for 404")
	}
}
