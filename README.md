# uber-exporter

Export your Uber trip receipt PDFs automatically.

Connects to your email via IMAP, finds Uber receipt emails from the last
3 months, extracts PDF download links, and saves them locally as
`YYYY-MM-DD-uber-receipt.pdf`.

## Prerequisites

- **Go 1.25+** — [install](https://go.dev/dl/)
- **A Gmail account** (or any IMAP-capable email) that receives Uber receipts
- **An app password** for your email (Gmail requires this if you have 2FA)
- **Your Uber browser cookies** (needed to download the actual PDF files)

## Setup

### Step 1: Clone and build

```bash
git clone https://github.com/zerotohero-dev/uber-exporter.git
cd uber-exporter
make build
```

This produces a `uber-exporter` binary in the project root.

### Step 2: Create the config directory

```bash
mkdir -p ~/.config/uber-exporter
```

### Step 3: Create the config file

```bash
cat > ~/.config/uber-exporter/config.json << 'EOF'
{
  "imap": {
    "server": "imap.gmail.com",
    "port": 993,
    "username": "you@gmail.com",
    "password_cmd": "cat ~/.config/uber-exporter/app-password.txt"
  },
  "outbox_dir": "outbox"
}
EOF
```

Edit `username` to your actual email address.

> **What is `password_cmd`?** It's a shell command that prints your IMAP
> password to stdout. This avoids storing the password in plain text in
> the config file. Some examples:
>
> | Method | `password_cmd` value |
> |--------|---------------------|
> | Plain text file | `cat ~/.config/uber-exporter/app-password.txt` |
> | macOS Keychain | `security find-generic-password -s uber-exporter -w` |
> | 1Password CLI | `op read "op://Personal/Gmail App Password/password"` |
> | pass | `pass show email/gmail-app-password` |

### Step 4: Store your email password

For Gmail, [create an app password](https://myaccount.google.com/apppasswords),
then save it:

```bash
echo "your-app-password-here" > ~/.config/uber-exporter/app-password.txt
chmod 600 ~/.config/uber-exporter/app-password.txt
```

### Step 5: Get your Uber cookies

The PDF download requires authentication with Uber. You need to export
your browser cookies after logging into [riders.uber.com](https://riders.uber.com).

1. Log into [riders.uber.com](https://riders.uber.com) in your browser
2. Open DevTools (F12) → Network tab
3. Click on any request to `riders.uber.com`
4. Find the `Cookie` header in the request headers
5. Copy the entire value and save it:

```bash
cat > ~/.config/uber-exporter/cookie.txt << 'EOF'
sid=YOUR_SID_VALUE; jwt-session=YOUR_JWT_VALUE; csid=YOUR_CSID_VALUE; udi-id=YOUR_UDI_VALUE
EOF
chmod 600 ~/.config/uber-exporter/cookie.txt
```

The cookie is a single line of `key=value` pairs separated by `; `.
The exact keys vary, but it typically includes `sid`, `jwt-session`,
`csid`, and `udi-id` among others. Copy the whole thing — don't try
to pick individual cookies.

> **Cookies expire.** If downloads start failing with "got HTML instead
> of PDF", your cookies have expired. Repeat this step.

### Step 6: Run it

```bash
make run
```

Or run the binary directly:

```bash
./uber-exporter
```

PDFs are saved to `outbox/` (or whatever you set `outbox_dir` to).
A log file `uber-exporter-YYYY-MM-DD-HHMMSS.log` is written to the
current directory.

## Config Reference

`~/.config/uber-exporter/config.json`:

```json
{
  "imap": {
    "server": "imap.gmail.com",
    "port": 993,
    "username": "you@gmail.com",
    "password_cmd": "cat ~/.config/uber-exporter/app-password.txt"
  },
  "outbox_dir": "outbox"
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `imap.server` | string | `imap.gmail.com` | IMAP server hostname |
| `imap.port` | int | `993` | IMAP TLS port |
| `imap.username` | string | *(required)* | Your email address |
| `imap.password_cmd` | string | *(required)* | Shell command that prints IMAP password |
| `outbox_dir` | string | `outbox` | Directory for downloaded PDFs |

`~/.config/uber-exporter/cookie.txt`:

A single line containing your Uber browser cookies (see Step 5 above).

## Troubleshooting

**"IMAP username not configured"** — You haven't created the config file
or `username` is empty. See Step 3.

**"no password_cmd configured"** — Add `password_cmd` to your config.
See Step 3.

**"IMAP login: … AUTHENTICATIONFAILED"** — Wrong password or you need
a Gmail app password (not your regular password). See Step 4.

**"got HTML instead of PDF"** — Your Uber cookies have expired.
Re-export them from your browser. See Step 5.

**"No Uber receipt emails found"** — The tool searches the last 3 months
of `[Gmail]/All Mail` for emails from `uber.com` containing "receipt".
Make sure your Uber receipt emails exist in that timeframe.

## License

PUBLIC DOMAIN

Use it however you like.
