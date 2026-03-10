# CanalBox SDK

## Overview

This is a Go SDK for interacting with the Salesforce Experience Cloud portal (Vivendi Africa distributor portal). It provides authentication and subscription management via Aura endpoints without OAuth.

## Renewal Flow

```go
ctx := context.Background()

client, err := canalbox.Login(baseURL, username, password)
if err != nil {
    return err
}

options, err := client.GetRenewOptionsByBox(ctx, boxNumber)
if err != nil {
    return err
}

offer := options.Offers[0].Name
preview, err := client.PreviewRenewByBox(ctx, boxNumber, offer, 1)
if err != nil {
    return err
}

if !preview.Success {
    for _, reason := range preview.Reasons {
        fmt.Println(reason.Message)
    }
}

activation, err := client.ActivateRenewByBox(ctx, boxNumber, offer, 1)
if err != nil {
    return err
}

if !activation.Success {
    for _, reason := range activation.Reasons {
        fmt.Println(reason.Message)
    }
}
```

`PreviewRenewByBox` and `ActivateRenewByBox` parse Salesforce nested `returnValue` payloads and expose business errors via `Reasons` (for example: not enough credit).

## Build & Test

### Build

```bash
# Build the SDK package
go build ./...

# Build example
cd example && go build -o ../canalbox .
```

### Run Example
```bash
./canalbox
```

### Testing
```bash
# Run all tests
go test ./...

# Run a single test
go test -run TestName ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

### Linting
```bash
# Run go vet
go vet ./...

# Run golint (if installed)
golangci-lint run

# Format code
go fmt ./...
```

## Code Style Guidelines

### General Principles

- **No unnecessary comments** - Code should be self-documenting
- **No TODO comments** unless they contain specific action items with ticket references
- **No AI-generated artifacts** - Avoid generic comments like "Create user" or "Save to database"
- **Keep functions small** - Aim for <50 lines per function
- **Single responsibility** - Each function should do one thing well

### Imports

Group imports in the following order (standard Go convention):
1. Standard library (fmt, os, io, etc.)
2. Third-party packages (crypto, net, etc.)
3. External packages (this project modules)

```go
import (
    "crypto/tls"
    "fmt"
    "io"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "strings"
    "time"

    "github.com/quarksgroup/canalbox-sdk"
)
```

### Formatting

- Use Go's standard formatting (run `go fmt` before committing)
- Use tabs for indentation, not spaces
- No trailing whitespace
- Maximum line length: 100 characters (soft limit)

### Naming Conventions

#### Variables & Functions
- Use camelCase for variables and functions
- Use PascalCase for exported types and functions
- Use short, descriptive names (e.g., `req`, `resp`, `cfg`)
- Avoid abbreviations unless widely understood (e.g., `cfg` is fine, `msg` is not)

#### Types
- Use meaningful type names (e.g., `Subscription`, not `Sub`)
- Suffix with appropriate type: `Client`, `Config`, `Service`
- Use interfaces when multiple implementations are possible

#### Constants
- Use PascalCase for exported constants
- Use camelCase for unexported constants
- Group related constants

```go
const DefaultBaseURL = "https://grpvivendiafrica.my.site.com"
```

### Error Handling

- Always handle errors explicitly with `if err != nil`
- Use `fmt.Errorf` with `%w` for wrapped errors
- Return meaningful error messages (lowercase, no punctuation)
- Check errors at the point of occurrence

```go
req, err := http.NewRequest(http.MethodGet, loginURL, nil)
if err != nil {
    return nil, fmt.Errorf("create login request: %w", err)
}
```

### HTTP Client Configuration

- Always set `ForceAttemptHTTP2: false` for Salesforce Aura endpoints
- Use cookiejar for session management
- Set appropriate timeouts (30 seconds is standard)

```go
client := &http.Client{
    Timeout: 30 * time.Second,
    Jar:     jar,
    Transport: &http.Transport{
        TLSClientConfig:   &tls.Config{},
        ForceAttemptHTTP2: false,
    },
}
```

### JSON Handling

- Use explicit struct tags for JSON fields
- Use `any` (interface{}) for dynamic response fields in Go 1.18+
- Use type assertions with ok idiom for safe type conversion

```go
if accountR, ok := m["Zuora__Account__r"].(map[string]any); ok {
    sub.Zuora__Account__r = Account{
        Phone: toString(accountR["Phone"]),
    }
}
```

### Response Parsing

- Create helper functions for common conversions (e.g., `toString`)
- Always close response bodies with `defer resp.Body.Close()`
- Use `io.Copy(io.Discard, resp.Body)` to drain unused bodies

### Cookie Handling

- Salesforce uses dynamic cookie names (e.g., `__Host-ERIC_PROD-*`)
- Use cookiejar for automatic cookie management
- Extract cookies after each request using `client.Jar.Cookies(url)`

```go
rawCookies := fmt.Sprintf("BrowserId=%s; oid=%s; sid=%s",
    c.cfg.BrowserID,
    c.cfg.OrgID,
    c.cfg.SID,
)
req.Header.Set("Cookie", rawCookies)
```

### Package Structure

```
canalbox-sdk/
├── client.go        # Client & Config structs, HTTP client setup
├── auth.go          # Authentication (Login function)
├── subscription.go  # Subscription types and methods
├── example/
│   └── main.go      # Usage example
└── go.mod
```

### Testing Guidelines

- Write tests in `*_test.go` files
- Use table-driven tests when testing multiple inputs
- Mock external dependencies when possible
- Test error cases, not just happy path

### Security

- Never log or expose credentials
- Use environment variables for sensitive configuration
- Validate all inputs before use

### Pull Request Guidelines

1. Run `go fmt` and `go vet` before submitting
2. Ensure code compiles with `go build ./...`
3. Run tests with `go test ./...`
4. Keep changes focused and minimal
5. Update this AGENTS.md if adding new patterns
