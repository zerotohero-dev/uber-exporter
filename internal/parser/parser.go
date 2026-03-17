package parser

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
	"golang.org/x/net/html"
)

type UberReceipt struct {
	Subject   string
	Date      time.Time
	PDFLink   string
	From      string
	MessageID string
}

// ParseEmail extracts the trip date and PDF download link from a raw Uber receipt email.
func ParseEmail(raw []byte) (*UberReceipt, error) {
	r, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("creating mail reader: %w", err)
	}

	header := r.Header
	subject, _ := header.Subject()
	date, _ := header.Date()
	from, _ := header.AddressList("From")
	messageID, _ := header.MessageID()

	var fromAddr string
	if len(from) > 0 {
		fromAddr = from[0].Address
	}

	receipt := &UberReceipt{
		Subject:   subject,
		Date:      date,
		From:      fromAddr,
		MessageID: messageID,
	}

	// Walk MIME parts looking for HTML body
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := h.ContentType()
			if ct != "text/html" {
				continue
			}

			body, err := io.ReadAll(p.Body)
			if err != nil {
				continue
			}

			link := extractPDFLink(string(body))
			if link != "" {
				receipt.PDFLink = link
				return receipt, nil
			}
		}
	}

	return receipt, nil
}

// extractPDFLink finds the receipt download URL in the email HTML.
// Uber emails use tracking redirects (click.uber.com, tracking.ibt.uber.com, etc.)
// that resolve to riders.uber.com/trips/<uuid>/receipt?contentType=PDF.
func extractPDFLink(htmlBody string) string {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return fallbackRegexExtract(htmlBody)
	}

	var pdfLink string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if pdfLink != "" {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			href := getAttr(n, "href")
			if href == "" {
				return
			}

			linkText := extractText(n)
			lowerText := strings.ToLower(linkText)

			// Match links whose text suggests a receipt/PDF download
			if matchesReceiptLink(lowerText) {
				pdfLink = href
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	if pdfLink != "" {
		return pdfLink
	}

	return fallbackRegexExtract(htmlBody)
}

func matchesReceiptLink(linkText string) bool {
	patterns := []string{
		"download pdf",
		"download receipt",
		"view receipt",
		"pdf receipt",
		"trip receipt",
		"descargar pdf",
		"descargar recibo",
		"ver recibo",
		"receipt",
	}
	for _, pat := range patterns {
		if strings.Contains(linkText, pat) {
			return true
		}
	}
	return false
}

var receiptURLRegex = regexp.MustCompile(`https?://[^\s"'<>]+(?:receipt|/pdf/)[^\s"'<>]*`)

func fallbackRegexExtract(body string) string {
	match := receiptURLRegex.FindString(body)
	return match
}

func getAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func extractText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}
