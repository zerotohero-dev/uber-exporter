package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/zerotohero-dev/uber-exporter/internal/config"
	"github.com/zerotohero-dev/uber-exporter/internal/downloader"
	"github.com/zerotohero-dev/uber-exporter/internal/email"
	"github.com/zerotohero-dev/uber-exporter/internal/parser"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.IMAP.Username == "" {
		return fmt.Errorf("IMAP username not configured. Create config at ~/.config/uber-exporter/config.json")
	}

	// Set up log file alongside stdout
	logName := fmt.Sprintf("uber-exporter-%s.log", time.Now().Format("2006-01-02-150405"))
	logFile, err := os.Create(logName)
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}
	defer logFile.Close()

	w := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(w, "", 0)

	logger.Println("Fetching Uber receipt emails from the last 3 months...")
	rawEmails, err := email.FetchUberReceipts(cfg)
	if err != nil {
		return fmt.Errorf("fetching emails: %w", err)
	}

	if len(rawEmails) == 0 {
		logger.Println("No Uber receipt emails found.")
		return nil
	}

	// Dump first receipt email HTML for debugging
	debugDir := "debug"
	os.MkdirAll(debugDir, 0o755)
	dumped := 0
	for i, raw := range rawEmails {
		if dumped >= 3 {
			break
		}
		debugFile := filepath.Join(debugDir, fmt.Sprintf("email-%03d.eml", i))
		os.WriteFile(debugFile, raw.Body, 0o644)
		dumped++
	}
	logger.Printf("Dumped %d sample emails to %s/ for debugging\n", dumped, debugDir)

	logger.Printf("Parsing %d emails...\n", len(rawEmails))
	var receipts []*parser.UberReceipt
	for _, raw := range rawEmails {
		r, err := parser.ParseEmail(raw.Body)
		if err != nil {
			logger.Printf("  Warning: failed to parse email: %v\n", err)
			continue
		}
		receipts = append(receipts, r)
	}

	if len(receipts) == 0 {
		logger.Println("No receipts could be parsed.")
		return nil
	}

	logger.Printf("Found %d receipts with PDF links out of %d emails\n",
		countWithPDF(receipts), len(receipts))

	logger.Printf("Downloading PDFs to %s/\n", cfg.OutboxDir)
	if err := downloader.DownloadReceipts(receipts, cfg.OutboxDir, cfg.Cookie, logger); err != nil {
		return fmt.Errorf("downloading receipts: %w", err)
	}

	logger.Printf("Log written to %s\n", logName)
	logger.Println("Done.")
	return nil
}

func countWithPDF(receipts []*parser.UberReceipt) int {
	n := 0
	for _, r := range receipts {
		if r.PDFLink != "" {
			n++
		}
	}
	return n
}
