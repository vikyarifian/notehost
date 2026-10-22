# notehost

Notehost is a lightweight internal markdown note sharing web application designed for fast publishing of operational guides, IT cheat sheets, and team documentation within internal network environments without the overhead of heavy enterprise wiki systems.

## Features

- Markdown editor with live preview rendered via HTMX swaps
- Real-time full-text search across titles, content, and tags
- Department tagging (e.g., `IT`, `HR`, `Operasional`, `Finance`)
- PIN-based edit protection for critical operational guides
- Single binary Go executable with embedded assets and zero external runtime dependencies

## Tech Stack

- **Backend**: Go 1.26 (net/http, html/template)
- **Database**: SQLite3 (`modernc.org/sqlite` embedded)
- **Frontend**: HTMX v2 + custom minimal CSS

## Quick Start (Development)

### Prerequisites

- Go 1.26 or higher
- SQLite runtime (`sqlite3` CLI optional for inspection)

### Running Locally

```bash
# Clone repository
git clone git@git.company.co.id:it-ops/notehost.git
cd notehost

# Run with local database file
export NOTEHOST_DB="./notehost_dev.db"
export NOTEHOST_PORT="8080"
go run main.go
```

Open `http://localhost:8080` in your web browser.

## Deployment

Notehost is distributed as a single compiled Go binary and managed via systemd on internal Ubuntu Linux nodes in the Jakarta data center.

### Building for Production

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o notehost main.go
```

### Systemd Service Configuration

Place the binary in `/usr/local/bin/notehost` and create `/etc/systemd/system/notehost.service`:

```ini
[Unit]
Description=Notehost Operational Knowledge Base
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/var/lib/notehost
ExecStart=/usr/local/bin/notehost
Restart=always
RestartSec=5s
Environment=NOTEHOST_DB=/var/lib/notehost/notehost.db
Environment=NOTEHOST_PORT=8080

[Install]
WantedBy=multi-user.target
```

Ensure database directory permissions:

```bash
sudo mkdir -p /var/lib/notehost
sudo chown -R www-data:www-data /var/lib/notehost
sudo systemctl daemon-reload
sudo systemctl enable --now notehost
```

### Backup

Daily automated filesystem backups snapshot `/var/lib/notehost/notehost.db` to the local internal backup storage server via cron.
