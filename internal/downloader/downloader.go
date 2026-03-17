package downloader

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/zerotohero-dev/uber-exporter/internal/parser"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/144.0.0.0 Safari/537.36"

// newClient creates an HTTP client that preserves the cookie header across redirects.
func newClient(cookie string) *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if cookie != "" {
				req.Header.Set("Cookie", cookie)
			}
			req.Header.Set("User-Agent", userAgent)
			return nil
		},
	}
}

// DownloadReceipts downloads PDF receipts for parsed Uber emails into the outbox directory.
// Files are named by trip date: YYYY-MM-DD-uber-receipt.pdf
// Duplicates on the same date get a suffix: YYYY-MM-DD-uber-receipt-2.pdf
func DownloadReceipts(receipts []*parser.UberReceipt, outboxDir string, cookie string, logger *log.Logger) error {
	if err := os.MkdirAll(outboxDir, 0o755); err != nil {
		return fmt.Errorf("creating outbox directory: %w", err)
	}

	client := newClient(cookie)
	seen := make(map[string]int)

	for _, r := range receipts {
		dateStr := r.Date.Format("2006-01-02")

		if r.PDFLink == "" {
			logger.Printf("  Skipping (no PDF link): %s [%s]\n", r.Subject, dateStr)
			continue
		}

		seen[dateStr]++

		filename := dateStr + "-uber-receipt.pdf"
		if seen[dateStr] > 1 {
			filename = fmt.Sprintf("%s-uber-receipt-%d.pdf", dateStr, seen[dateStr])
		}

		dest := filepath.Join(outboxDir, filename)

		if _, err := os.Stat(dest); err == nil {
			logger.Printf("  Already exists: %s\n", filename)
			continue
		}

		logger.Printf("  Downloading: %s\n", filename)
		logger.Printf("    URL: %s\n", r.PDFLink)
		if err := downloadFile(client, r.PDFLink, dest, cookie, logger); err != nil {
			logger.Printf("    Error: %v\n", err)
			continue
		}
		logger.Printf("    Saved: %s\n", dest)
	}

	return nil
}

func downloadFile(client *http.Client, url, dest string, cookie string, logger *log.Logger) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP GET: %w", err)
	}
	defer resp.Body.Close()

	logger.Printf("    Final URL: %s", resp.Request.URL.String())
	logger.Printf("    Response: %d %s", resp.StatusCode, resp.Status)
	logger.Printf("    Content-Type: %s", resp.Header.Get("Content-Type"))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Verify we got a PDF (or at least not an HTML error page)
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		return fmt.Errorf("got HTML instead of PDF (may need browser auth)")
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, resp.Body)
	if err != nil {
		os.Remove(dest)
		return fmt.Errorf("writing file: %w", err)
	}
	logger.Printf("    Size: %d bytes", n)

	return nil
}
