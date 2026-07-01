# go-9router

Lightweight, high-performance Go rewrite of the `9router` project, optimized **specifically** and exclusively for the **Antigravity** provider proxying over SSE.

It accepts standard OpenAI API requests (e.g. Chat Completions) and proxies/translates them into Gemini/Antigravity format, and translates the stream response chunk back to standard OpenAI Server-Sent Events (SSE).

## Features
- Super lightweight: Extremely low CPU & memory consumption (compiled Go binary).
- Translates OpenAI request format to Antigravity format.
- Fully supports SSE stream translation on-the-fly (`/v1/chat/completions`).
- Supports non-streaming responses.
- Auto OAuth Token Pre-warming & refreshing.
- Proxy support (HTTP/HTTPS/SOCKS5).
- Rate limit (429) & backoff parsing.
- Optional API key protection.

## Getting Started

### Prerequisites
- Go 1.20+

### Configuration Options
You can configure the application using either **CLI Flags** (recommended) or **Environment Variables**. CLI Flags take precedence.

| CLI Flag | Env Variable | Description |
|---|---|---|
| `-port` | `PORT` | Port to run the server on (default: `20128`) |
| `-user` | `AUTH_USER` | Username for Basic Auth |
| `-password` | `AUTH_PASSWORD` | Password for Basic Auth |
| `-api-key` | `API_KEY` | Optional Bearer API Key fallback |
| `-refresh-token` | `REFRESH_TOKEN` | Google OAuth Refresh Token |
| `-client-id` | `CLIENT_ID` | Google Client ID |
| `-client-secret` | `CLIENT_SECRET` | Google Client Secret |
| `-proxy` | `PROXY_URL` | SOCKS5 or HTTP proxy URL |
| `-proxy-pool` | `PROXY_POOL_URLS` | Comma-separated list of proxies (e.g. `http://p1:8080,socks5://p2:1080`) |
| `-vercel-relay` | `VERCEL_RELAY_URL` | Vercel serverless function relay URL |
| `-mitm` | | Auto-start MITM Server on startup (port 443, requires sudo/admin) |

## Standalone Web UI Dashboard

The Go rewrite includes a built-in, lightweight Standalone Web UI Dashboard. It is compiled directly into the binary using Go's `//go:embed` feature, allowing `go-9router` to run with zero external assets or Node.js dependencies.

* **Access URL**: Open your browser at `http://localhost:<port>/` (default: `http://localhost:20128/`).
* **Network Access**: The server automatically binds to `0.0.0.0` (all network interfaces), meaning you can access the dashboard and connect client APIs (like Cursor) from any device/IP on your local network.
* **Visual Status Overview**: Live status cards monitoring the Router Server, HTTPS MITM proxy, and Hosts DNS overrides.
- **MITM Control Panel**: Toggle the HTTPS proxy server and system-wide local DNS poisoning (requires sudo password on Unix systems or Admin privilege on Windows).
- **Backup & Settings Management**: Buttons to download settings database backup files (`GET /api/settings/database`) and import them (`POST /api/settings/database`).
- **Live Activity Logs**: An integrated terminal console emulator displaying the logs tailing `/api/translator/console-logs` dynamically with automatic color highlighting based on level.

---

### Proxy Pools (Rotasi Proxy)
Jika Anda mendefinisikan beberapa proxy lewat flag `-proxy-pool` (atau env `PROXY_POOL_URLS`), aplikasi akan mem-proxy outgoing request ke Google/Antigravity secara bergantian (*Round-Robin*) secara thread-safe dan dinamis untuk setiap request-nya.

### Vercel Relay
Jika flag `-vercel-relay` (atau env `VERCEL_RELAY_URL`) dikonfigurasi, aplikasi akan membungkus request keluar ke Google Cloud Code melalui Vercel serverless relay:
- Menyuntikkan header `x-relay-target: https://daily-cloudcode-pa.googleapis.com`
- Menyuntikkan header `x-relay-path: /v1internal:...`
- Mengarahkan destination URL request ke Vercel Relay Anda.

*(Catatan: Refresh Token OAuth akan tetap dikirim secara direct ke API Google untuk keamanan dan menghindari kegagalan token).*

### Installation & Run

1. Navigate to the directory:
   ```bash
   cd go-9router
   ```

2. Download dependencies:
   ```bash
   go mod tidy
   ```

3. Run the server using **CLI Flags** (tanpa export port / credentials):
   ```bash
   go run cmd/9router/main.go -port 20128 -user admin -password mypassword -refresh-token xxx -client-id yyy -client-secret zzz
   ```

4. Run the server WITH MITM Server activated (requires `sudo` for port 443):
   ```bash
   sudo go run cmd/9router/main.go -port 20128 -mitm -refresh-token xxx -client-id yyy -client-secret zzz
   ```

5. Build binary:
   ```bash
   go build -o 9router cmd/9router/main.go
   ./9router -port 20128 -user admin -password mypassword
   ```

## MITM Capabilities & Control API

The application ports the control endpoints from the original 9router Node dashboard to handle dynamic starting/stopping and DNS poisoning of Google Antigravity endpoints:

- `GET /api/cli-tools/antigravity-mitm`: Returns status (`running` and `dns` status).
- `POST /api/cli-tools/antigravity-mitm`: Controls the MITM server. Accept JSON:
  - `{"action": "start"}`: Starts the MITM server on port 443.
  - `{"action": "stop"}`: Stops the MITM server.
  - `{"action": "dns_enable", "password": "optional_sudo_password"}`: Redirects `cloudcode-pa.googleapis.com` to `127.0.0.1` in hosts file.
  - `{"action": "dns_disable", "password": "optional_sudo_password"}`: Removes the DNS redirection in hosts file.

## Backup & Restore Settings API (Download & Import Backup)

To align with the database export/import functionality in the dashboard settings menu, the Go app implements a JSON-file-based persistent database mimicking the exact Node.js SQLite export data format:

- **Storage Location**: Saved locally in platform-specific standard directories:
  - **Windows**: `%APPDATA%/9router/db/data.json`
  - **macOS/Linux**: `~/.9router/db/data.json`
- **One-Time Auto-Migration**: If the Go app detects an existing Node.js SQLite database (`data.sqlite`) in the database directory, and the JSON file (`data.json`) does not exist yet, it will **automatically read the SQLite file, convert all configuration tables, and migrate them** to the new JSON file format on boot. The SQLite database is then renamed to `data.sqlite.migrated` to mark completion.
- **Export Backup (`GET /api/settings/database`)**: Retrieves the complete JSON representation of the database.
- **Import Backup (`POST /api/settings/database`)**: Overwrites the local JSON file database with the imported JSON configuration.
- **Dynamic Configuration reloading**: Upon import, the Go backend automatically parses the credentials (refresh tokens, client IDs) and proxies (proxy pool connections) stored in the backup and **reloads the dependencies dynamically** on-the-fly without requiring a restart of the binary.

## Quota Tracking & Connection Lock Mode

The Go rewrite brings full lock mode support and token usage tracking directly mapped into the database backup format:

- **Lock Mode (Temporary Cooldown)**: When a request hits a 429 Rate Limit error from Google/Antigravity API, the reset time (Retry-After) is parsed. If the reset delay is long (or retries are exhausted), the connection is locked for that specific model and its associated quota family (`ag-gemini` for Gemini models, `ag-non-gemini` for others). Further requests targeting that connection/model will be rejected locally until the cooldown expires.
- **Usage Tracker**: Successfully completed chat completions (both streaming and non-streaming) automatically calculate the prompt & completion tokens consumed, incrementing the connection usage statistics (`requestsLifetime`, `promptTokensLifetime`, `completionTokensLifetime`) directly inside the local `data.json` database.

---

## API Endpoint Usage

### Health Check
```bash
GET /health
```

### Chat Completions (OpenAI Compatible)
```bash
POST /v1/chat/completions
```

#### Example cURL Request with Basic Auth:
```bash
curl http://localhost:20128/v1/chat/completions \
  -u "admin:mypassword" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "ag",
    "messages": [{"role": "user", "content": "Hello! How are you?"}],
    "stream": true
  }'
```

#### Example cURL Request with API Key (Bearer):
```bash
curl http://localhost:20128/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-optional-9router-api-key" \
  -d '{
    "model": "ag",
    "messages": [{"role": "user", "content": "Hello! How are you?"}],
    "stream": true
  }'
```
