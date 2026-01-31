# AGENTS.md - Guidelines for Agentic Coding Agents

This document provides guidelines for agents working on the Gwxapkg codebase.

## Project Overview

Gwxapkg is a Go-based tool for unpacking WeChat Mini Program packages (.wxapkg). It supports automatic scanning, decryption, decompilation, and security analysis. The project uses Go 1.23+ and follows standard Go conventions.

## Build Commands

```bash
# Build optimized binary
go build -ldflags="-s -w" -o gwxapkg .

# Build for specific platforms
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o gwxapkg-darwin .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o gwxapkg.exe .

# Run directly
go run . -h

# Run all tests
go test ./... -v

# Run single test
go test -v ./internal/scanner/... -run TestFunctionName

# Update dependencies
go mod tidy
```

## Code Style Guidelines

### General Principles

- Follow standard Go conventions (Effective Go, Go Code Review Comments)
- Keep functions short and focused (ideally < 50 lines)
- Use meaningful variable and function names
- Avoid unnecessary comments; code should be self-documenting
- Prefer early returns and guard clauses over deeply nested code

### Naming Conventions

- **Packages**: Short, lowercase names (`scanner`, `decrypt`, `formatter`)
- **Files**: snake_case (`sensitive_filter.go`)
- **Variables**: camelCase (`appID`, `outputDir`, `fileExt`)
- **Types/Interfaces**: PascalCase (`SensitiveItem`, `Reader`)
- **Error variables**: Prefix with `Err` or `err` (`ErrInvalidInput`)
- **Acronyms**: Keep original casing (`appID`, not `appId`)

### Imports

Group imports in order: (1) standard library, (2) third-party, (3) internal packages.
```go
import (
    "bufio"
    "fmt"
    "path/filepath"
    "sync"
    "sync/atomic"

    "github.com/25smoking/Gwxapkg/internal/key"
    "github.com/25smoking/Gwxapkg/internal/ui"
)
```

### Formatting

- Use `gofmt` (default formatting), indent with tabs
- No trailing whitespace, blank line between top-level declarations
- One space after commas, around operators

### Types and Declarations

- Use `var` at package level for zero-value initialization, `:=` for local
- Group related declarations:
  ```go
  var (
      fileNameBlacklist = map[string]bool{...}
      fileExtPattern    = regexp.MustCompile(...)
  )
  ```
- Use constants for magic numbers, `iota` for enum-like constants

### Error Handling

- Handle errors explicitly, return meaningful errors with context:
  ```go
  return nil, fmt.Errorf("error reading rule file: %v", err)
  ```
- Check errors immediately after calls, use `errors.Wrap` for stack traces

### Concurrency

- Use goroutines for concurrent operations, `sync.WaitGroup` for joining
- Use channels for communication, `sync.Mutex`/`RWMutex` for shared state
- Prefer `context.Context` for cancellation, `atomic` for simple counters

### Testing

- Write tests for new functions (`filename_test.go`)
- Use table-driven tests:
  ```go
  tests := []struct {
      name    string
      input   string
      want    string
  }{{"case1", "input", "expected"}}
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { /* test logic */ })
  }
  ```

### Project Structure

```
Gwxapkg/
├── cmd/              # CLI entry point
├── internal/
│   ├── cmd/          # Command processing
│   ├── decrypt/      # AES+XOR decryption
│   ├── unpack/       # wxapkg binary parsing
│   ├── restore/      # Project structure restoration
│   ├── formatter/    # Code beautification
│   ├── key/          # Rule management, pre-compilation
│   ├── scanner/      # Sensitive information scanning
│   ├── reporter/     # Excel report generation
│   ├── config/       # Configuration management
│   ├── locator/      # WeChat cache location finder
│   ├── ui/           # Terminal UI and progress bars
│   ├── pack/         # Repack functionality
│   └── util/         # Utility functions
├── config/           # YAML rule files
└── main.go           # Entry point
```

### Key Packages

- `github.com/tdewolff/parse/v2` - Parser utilities
- `github.com/dop251/goja` - JavaScript engine
- `github.com/ditashi/jsbeautifier-go` - JS beautifier
- `github.com/yosssi/gohtml` - HTML formatter
- `golang.org/x/crypto` - Cryptographic functions
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/xuri/excelize/v2` - Excel generation

### Performance Considerations

- Pre-compile regex patterns at initialization (`key.InitRules()`)
- Use buffered I/O (256KB buffer for file operations)
- Dynamic concurrency based on CPU cores (`runtime.NumCPU()`)
- Reuse buffers to reduce allocations

### Security Guidelines

- This tool is for security research on WeChat Mini Programs
- Do not introduce code that exposes secrets or keys
- Do not commit credentials, tokens, or API keys
- Follow existing filter patterns in `internal/scanner/filter.go`

### Common Tasks

**Add a new scanning rule:**
1. Add rule to `internal/key/key.go` `CreateConfigFile()`
2. Add name mapping in `internal/scanner/scanner.go` `getRuleName()`
3. Add false positive patterns to `internal/scanner/filter.go` if needed

**Add a new file type:**
1. Add parser in `internal/unpack/` (e.g., `uxml.go`, `ujs.go`)
2. Add formatter in `internal/formatter/` if needed
3. Update `internal/restore/` for structure restoration

**Modify UI output:**
1. Edit `internal/ui/ui.go` for banner, messages, progress bars
2. Use functions: `ui.Info()`, `ui.Success()`, `ui.Warning()`, `ui.Error()`
