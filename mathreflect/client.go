package mathreflect

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// DefaultUserAgent identifies the client to awesomemath.org.
const DefaultUserAgent = "mathreflect/dev (+https://github.com/tamnd/mathreflect-cli)"

// rePDFURL matches problem PDF URLs in the archive page.
var rePDFURL = regexp.MustCompile(`/wp-pdf-files/math-reflections/(mr-\d{4}-\d{2})/mr_\d+_\d+_problems[^"']*\.pdf`)

// reIssueYearNum extracts year and issue number from a PDF URL folder segment.
// Example: mr-2025-06 -> year=2025, num=6
var reIssueFolder = regexp.MustCompile(`mr-(\d{4})-(\d{2})`)

// Client makes HTTP requests to awesomemath.org with pacing and retries.
type Client struct {
	http  *http.Client
	delay time.Duration
	mu    sync.Mutex
	last  time.Time
}

// NewClient returns a Client with the given delay and timeout.
func NewClient(delay, timeout time.Duration) *Client {
	return &Client{
		http: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:    10,
				IdleConnTimeout: 90 * time.Second,
			},
		},
		delay: delay,
	}
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.delay - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func (c *Client) get(ctx context.Context, url string) ([]byte, int, error) {
	var lastErr error
	for attempt := 0; attempt <= 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, 0, ctx.Err()
			case <-time.After(backoffDur(attempt)):
			}
		}
		body, code, retry, err := c.do(ctx, url)
		if err == nil {
			return body, code, nil
		}
		lastErr = err
		if !retry {
			return nil, code, err
		}
	}
	return nil, 0, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) (body []byte, code int, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, false, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, true, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, resp.StatusCode, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, 404, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, resp.StatusCode, true, err
	}
	return b, resp.StatusCode, false, nil
}

func backoffDur(attempt int) time.Duration {
	d := time.Duration(1<<uint(attempt-1)) * time.Second
	if d > 8*time.Second {
		d = 8 * time.Second
	}
	return d
}

// ArchiveResult holds problem PDF URLs from the archive page.
type ArchiveResult struct {
	PDFURLs []string
}

// FetchArchiveIndex fetches the archive page and extracts problem PDF URLs.
func (c *Client) FetchArchiveIndex(ctx context.Context) (*ArchiveResult, error) {
	body, code, err := c.get(ctx, ArchiveURL)
	if err != nil {
		return nil, fmt.Errorf("fetch archive: %w", err)
	}
	if code != 200 && code != 0 {
		return nil, fmt.Errorf("archive page returned HTTP %d", code)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse archive HTML: %w", err)
	}

	seen := make(map[string]bool)
	var result ArchiveResult

	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !rePDFURL.MatchString(href) {
			return
		}
		fullURL := href
		if !strings.HasPrefix(href, "http") {
			fullURL = "https://www.awesomemath.org" + href
		}
		if !seen[fullURL] {
			seen[fullURL] = true
			result.PDFURLs = append(result.PDFURLs, fullURL)
		}
	})

	return &result, nil
}

// FetchPDF downloads a PDF and returns its bytes. Returns (nil, 404, nil) for 404s.
func (c *Client) FetchPDF(ctx context.Context, url string) ([]byte, int, error) {
	return c.get(ctx, url)
}

// HeadURL issues a HEAD request and returns the status code.
func (c *Client) HeadURL(ctx context.Context, url string) (int, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", DefaultUserAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

// ExtractIssueYearNum parses year and issue number from a PDF URL.
// Example: .../mr-2025-06/mr_6_2025_problems.pdf -> (2025, 6)
func ExtractIssueYearNum(pdfURL string) (year, num int) {
	m := reIssueFolder.FindStringSubmatch(pdfURL)
	if m == nil {
		return 0, 0
	}
	// m[1]=year, m[2]=issue (zero-padded)
	fmt.Sscanf(m[1], "%d", &year)
	fmt.Sscanf(m[2], "%d", &num)
	return year, num
}
