# QuoteBot

A bot that periodically posts quotes to Bluesky

## Requirements

- Go 1.21 or higher
- Bluesky account

## Environment Variables

The following environment variables need to be set:

### Required Environment Variables

| Environment Variable | Description | Example |
|----------|------|-----|
| `ACCESS_JWT` | Bluesky access token | `eyJ0eXAiOi...` |
| `REFRESH_JWT` | Bluesky refresh token | `eyJ0eXAiOi...` |
| `DID` | Bluesky DID | `did:plc:...` |

### Optional Environment Variables

| Environment Variable | Description | Default Value |
|----------|------|------------|
| `PDS_URL` | Bluesky PDS URL | `https://bsky.social` |
| `QUOTES_FILE` | JSON file for quote data | `quotes.json` |
| `POST_INTERVAL` | Posting interval (e.g., 30m, 1h, 2h) | `1h` |
| `HTTP_TIMEOUT` | HTTP request timeout | `10s` |

## Setting Environment Variables

### Unix/Linux/macOS

```bash
# Required environment variables
export ACCESS_JWT="your_access_jwt"
export REFRESH_JWT="your_refresh_jwt"
export DID="your_did"

# Optional environment variables (if needed)
export PDS_URL="https://bsky.social"
export QUOTES_FILE="quotes.json"
export POST_INTERVAL="1h"
export HTTP_TIMEOUT="10s"
```

### Windows (PowerShell)

```powershell
# Required environment variables
$env:ACCESS_JWT="your_access_jwt"
$env:REFRESH_JWT="your_refresh_jwt"
$env:DID="your_did"

# Optional environment variables (if needed)
$env:PDS_URL="https://bsky.social"
$env:QUOTES_FILE="quotes.json"
$env:POST_INTERVAL="1h"
$env:HTTP_TIMEOUT="10s"
```

## How to Get Bluesky Tokens

1. Log in to https://bsky.app
2. Open developer tools (Press `F12` or `Command + Option + I` in Chrome/Safari)
3. Select the `Application` tab
4. Select `bsky.social` from `Local Storage` on the left
5. Copy the following values:
   - `did`: Your DID
   - `jwt`: Access token (ACCESS_JWT)
   - `refreshJwt`: Refresh token (REFRESH_JWT)

## Project Structure

```
.
├── main.go                 # Entry point
├── config/                 # Configuration
├── internal/               # Internal packages
│   ├── domain/            # Domain logic
│   ├── usecase/           # Use cases
│   └── interface/         # Interfaces
│       └── repository/    # Repository implementations
└── quotes.json            # Quote data
```

## Build and Run

```bash
# Build
go build -o quotebot

# Run
./quotebot
```

## Features

- Automatically posts quotes at set intervals (default: 1 hour)
- Automatic access token renewal
- Automatic retry on errors
- Customizable posting interval
- Quote posting functionality

## Troubleshooting

Common errors and solutions:

1. `required key XXX missing value`
   - Required environment variables are not set
   - Please set all required environment variables listed above

2. `failed to refresh token`
   - Token refresh failed
   - Verify that ACCESS_JWT and REFRESH_JWT are correct
   - If tokens are expired, obtain new ones

3. `failed to post message`
   - Posting failed
   - Check your internet connection
   - Verify that your tokens are valid

## Security Notes

- Tokens (ACCESS_JWT, REFRESH_JWT) are sensitive information. Do not share them with others
- Do not store tokens in source code or Git repositories
- It is recommended to update tokens periodically

## License

MIT
