package email

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"

	"github.com/zerotohero-dev/uber-exporter/internal/config"
)

type RawEmail struct {
	UID  imap.UID
	Body []byte
}

// FetchUberReceipts connects to Gmail via IMAP, searches All Mail for
// Uber receipt emails from the last 3 months, and returns their raw bodies.
func FetchUberReceipts(cfg config.Config) ([]RawEmail, error) {
	password, err := resolvePassword(cfg.IMAP.PasswordCmd)
	if err != nil {
		return nil, fmt.Errorf("resolving IMAP password: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", cfg.IMAP.Server, cfg.IMAP.Port)
	c, err := imapclient.DialTLS(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("connecting to IMAP server: %w", err)
	}
	defer c.Close()

	if err := c.Login(cfg.IMAP.Username, password).Wait(); err != nil {
		return nil, fmt.Errorf("IMAP login: %w", err)
	}

	// Use All Mail to find archived emails
	mailbox := "[Gmail]/All Mail"
	if _, err := c.Select(mailbox, nil).Wait(); err != nil {
		return nil, fmt.Errorf("selecting %s: %w", mailbox, err)
	}

	since := time.Now().AddDate(0, -3, 0)

	// Search for emails from uber.com in the last 3 months
	data, err := c.UIDSearch(&imap.SearchCriteria{
		Since: since,
		Header: []imap.SearchCriteriaHeaderField{
			{Key: "FROM", Value: "uber.com"},
		},
		Body: []string{"receipt"},
	}, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("searching for Uber receipts: %w", err)
	}

	uids := data.AllUIDs()
	if len(uids) == 0 {
		return nil, nil
	}

	fmt.Printf("Found %d Uber receipt emails\n", len(uids))

	uidSet := imap.UIDSet{}
	for _, uid := range uids {
		uidSet.AddNum(uid)
	}

	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{
			{}, // Full RFC822 message
		},
	}

	fetchCmd := c.Fetch(uidSet, fetchOptions)
	defer fetchCmd.Close()

	var emails []RawEmail

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		var body []byte
		for {
			item := msg.Next()
			if item == nil {
				break
			}
			if bodyData, ok := item.(imapclient.FetchItemDataBodySection); ok {
				buf := new(bytes.Buffer)
				if _, err := buf.ReadFrom(bodyData.Literal); err != nil {
					fmt.Printf("Warning: error reading email body: %v\n", err)
					continue
				}
				body = buf.Bytes()
			}
		}

		if len(body) == 0 {
			continue
		}

		emails = append(emails, RawEmail{Body: body})
	}

	if err := fetchCmd.Close(); err != nil {
		return emails, fmt.Errorf("fetch command: %w", err)
	}

	if err := c.Logout().Wait(); err != nil {
		return emails, fmt.Errorf("IMAP logout: %w", err)
	}

	return emails, nil
}

func resolvePassword(passwordCmd string) (string, error) {
	if passwordCmd == "" {
		return "", fmt.Errorf("no password_cmd configured")
	}

	cmd := exec.Command("sh", "-c", passwordCmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("running password command: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
