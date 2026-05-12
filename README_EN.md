# Gwxapkg

<div align="center">

![Version](https://img.shields.io/badge/version-2.7.3-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)
![Build](https://img.shields.io/badge/build-passing-brightgreen.svg)

**[中文](README.md) | [English](README_EN.md) | [日本語](README_JA.md)**

A Go-based WeChat Mini Program `.wxapkg` unpacker with automatic scanning, decryption, decompilation, and security analysis.

</div>

---

## ⚠️ Legal and Usage Notice

**This tool may only be used for security research, reverse engineering, compatibility verification, learning, and technical auditing of self-owned or entrusted assets, and only where such use is lawful, proper, and fully authorized.**

**Before using this tool, every user must independently confirm that they have clear, continuing, and demonstrable legal authorization over the target Mini Program, related accounts, devices, data, network environment, business systems, and any other associated resources. If you cannot confirm the scope or validity of such authorization, you must stop using this tool immediately.**

### Prohibited Uses

**This tool must not be used in any unauthorized scenario or in any way that may violate applicable laws, platform rules, contractual obligations, intellectual property rights, or privacy protections, including but not limited to:**

- **Unpacking, analyzing, extracting, copying, or distributing third-party Mini Program code, assets, or data without permission**
- **Bypassing platform protections, risk-control mechanisms, access restrictions, or any other technical controls**
- **Bulk collection, data scraping, privacy extraction, API abuse, or automated offensive testing**
- **Commercial theft, malicious imitation, malicious distribution, abuse in illicit industries, or any other improper purpose**
- **Any conduct that may cause harm to others, platform penalties, business interruption, data leakage, or compliance risks**

### Risk Notice

**Use of this tool may involve and result in risks including but not limited to legal liability, administrative penalties, civil damages, criminal exposure, intellectual property disputes, privacy infringement claims, account suspension, service interruption, data leakage, production incidents, and third-party claims.**

**Users are solely responsible for evaluating and bearing all risks and consequences arising from their use, misuse, or abuse of this tool.**

### Limitation of Liability

**To the maximum extent permitted by applicable law, this project and its authors, contributors, and maintainers shall not be liable for any direct, indirect, incidental, special, punitive, or consequential loss arising from the use, inability to use, misuse, or abuse of this tool, including but not limited to data loss, information leakage, privacy infringement, intellectual property disputes, platform penalties, account suspension, system failures, business loss, financial loss, administrative liability, or criminal liability.**

**This tool is provided on an “as is” basis, without any express or implied warranty, including but not limited to warranties of merchantability, fitness for a particular purpose, stability, accuracy, completeness, continued availability, or legal suitability.**

### Use Constitutes Acceptance

**By continuing to download, install, run, distribute, or use this tool, you acknowledge that you have read, understood, and agreed to this notice in full, and that you will use this tool only within a lawfully authorized scope.**

---

## ✨ Core Features

### Smart Unpacking
- **Automatic scanning** - Detects WeChat Mini Program cache directories on macOS and Windows
- **Automatic decryption** - Supports encrypted `wxapkg` decryption from desktop WeChat cache
- **One-command unpacking** - Finds and processes all packages for a specified AppID
- **Subpackage handling** - Correctly handles dependencies between main and subpackages

### Code Restoration
- **Full restoration** - Supports `wxml`, `wxss`, `js`, `json`, and `wxs`
- **Code beautification** - Automatically formats JavaScript, CSS, and HTML
- **Default deobfuscation** - Performs static restoration and controlled decoding for common string-array patterns, `\xNN`, `\uNNNN`, and hexadecimal literals
- **Source-level semantic restoration** - Enables AST `deep` mode by default to rewrite compressed local names into audit-friendly names such as `params`, `requestData`, `response`, `event`, and `app`
- **Traceable AST write-back** - Generates `ast_rename_map.json`, `ast_rename_diff.md`, and `ast_rename.patch`, while keeping pre-write sources for `semantic -ast-rollback=true`
- **API semantic views** - Generates `api_map`, API call chains, audit pseudocode, and can link raw Burp requests back to source APIs
- **Route map generation** - Builds page lists, entry pages, subpackages, TabBar pages, component dependencies, static and dynamic navigation edges, trigger clues, and page-level API attribution
- **Project structure restore** - Restores the original Mini Program directory structure
- **Asset extraction** - Fully extracts images, audio, video, and other resource files

### Security Analysis
- **Smart scanning** - 920 built-in sensitive-data detection rules enabled by default
- **Formal categories** - Rules are normalized into cloud, payment, collaboration, monitoring, security, SaaS, and other formal groups
- **False-positive filtering** - Blacklist plus placeholder, sample-value, masked-value, and weak-value filtering to reduce scan noise
- **Deduplication** - Removes duplicate findings and preserves precise locations
- **API extraction** - Extracts URL / API endpoints and can export a Postman Collection
- **Excel / HTML reports** - Generates professional multi-sheet Excel and interactive HTML reports with file paths and line numbers
- **Risk grading** - Automatically classifies high, medium, and low risk findings
- **Obfuscation reporting** - Lists obfuscated files and restoration status separately in reports

### Performance Optimization
- **Dynamic concurrency** - Adjusts worker count based on CPU cores
- **Buffered I/O** - Uses a 256 KB buffer to improve file read/write performance
- **Precompiled rules** - Compiles regex rules at startup to avoid repeated overhead
- **Optimized builds** - Uses optimized build flags for smaller binaries and faster execution

---

## 📊 Supported File Types

| File Type | Support | Description |
|-----------|---------|-------------|
| `.wxml` | Yes | Page structure restoration |
| `.wxss` | Yes | Style restoration |
| `.js` | Yes | JavaScript restoration, beautification, and default deobfuscation |
| `.json` | Yes | Configuration extraction |
| `.wxs` | Yes | WXS restoration |
| Images / Audio / Video | Yes | Full asset extraction |

---

## 📥 Installation

### Option 1: Download Prebuilt Binaries

Download the correct executable for your platform from [Releases](https://github.com/25smoking/Gwxapkg/releases).

### Option 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/25smoking/Gwxapkg.git
cd Gwxapkg

# Build an optimized binary
go build -ldflags="-s -w" -o gwxapkg .

# Or run directly
go run . -h
```

**Requirement:** Go 1.21 or later

---

## 🚀 Quick Start

### Basic Usage

```bash
# Automatically scan and process a Mini Program by AppID
./gwxapkg all -id=<AppID>

# List all available Mini Programs
./gwxapkg scan

# Show WeChat cache candidate-path diagnostics
./gwxapkg scan --verbose

# Unpack a single wxapkg file
./gwxapkg -id=<AppID> -in=<file_path>

# Scan an already unpacked directory and export Postman Collection
./gwxapkg scan-only -dir=<directory> -format=both -postman

# Repack
./gwxapkg repack -in=<directory_path>
```

### CLI Parameters

| Flag | Description | Default |
|------|-------------|---------|
| `-id` | Mini Program AppID | - |
| `-in` | Input file or directory | - |
| `-out` | Output directory | auto |
| `-restore` | Restore project directory structure | true |
| `-pretty` | Beautify output | true |
| `-sensitive` | Enable sensitive-data scanning | true |
| `-postman` | Export `api_collection.postman_collection.json` | false |
| `-noClean` | Keep intermediate temporary files | false |
| `-save` | Save decrypted files | false |
| `-workspace` | Keep hidden workspace for precise repacking | false |
| `--verbose` | Print cache candidate-path diagnostics (`scan` / `all` only) | false |

### Examples

```bash
# Example 1: auto scan and process by AppID
./gwxapkg all -id=wx3c19e32cb8f31289

# Example 2: unpack and export Postman Collection
./gwxapkg all -id=wx3c19e32cb8f31289 -postman

# Example 3: unpack a single file
./gwxapkg -id=wx123456 -in=test.wxapkg -out=./output

# Example 4: rescan an unpacked directory
./gwxapkg scan-only -dir=./output/wx123456 -format=both -postman

# Example 5: repack
./gwxapkg repack -in=./source_dir -out=new.wxapkg
```

### Default Output Directory

If `-out` is not specified, the output rules are:

- Compiled binary: `output/<AppID>` under the executable directory
- `go run .` or development mode: `output/<AppID>` under the current working directory
- Interactive `scan` mode: also uses `output/<AppID>`

Example:

```text
/Applications/Gwxapkg/output/wx1234567890abcdef
./output/wx1234567890abcdef
```

### Typical Output Structure

```text
output/
└── wx1234567890abcdef/
    ├── app.js
    ├── page-frame.html
    ├── sensitive_report.xlsx
    ├── sensitive_report.html
    ├── api_collection.postman_collection.json
    ├── route_manifest.json
    ├── route_map.md
    ├── route_map.mmd
    └── .gwxapkg/                   # only when -workspace=true
```

---

## 📁 WeChat Mini Program Cache Locations

### macOS

```text
~/Library/Containers/com.tencent.xinWeChat/Data/Library/Caches/
├── applet/
│   ├── release/
│   └── debug/
└── ...
```

### Windows

```text
%USERPROFILE%\Documents\WeChat Files\Applet\
├── wx<appid>/
│   ├── <version>/
│   │   ├── __APP__.wxapkg
│   │   └── __SUBCONTEXT__.wxapkg
│   └── ...
└── ...
```

---

## 🎯 Sensitive Information Scanning

### Scanning Rules (920 Built-in Rules)

920 built-in rules are enabled by default. Although many of the newly added rules still use names like `xxx secret`, the program no longer classifies them purely by literal names. Instead, it prioritizes service domain and usage so the report stays organized instead of dumping everything into `secret` or `other`.

| Category | Rules | Examples |
|----------|-------|----------|
| **Third-party SaaS** | 596 | Various `xxx secret`, service-specific tokens, webhooks, client credentials |
| **Secrets / Keys** | 67 | Generic `client_secret`, `app_secret`, `session_secret` |
| **Development & Delivery** | 40 | GitHub, GitLab, NPM, PyPI, Jenkins, Terraform, JFrog |
| **Cloud Platforms** | 35 | AWS, Alibaba Cloud, Tencent Cloud, Huawei Cloud, Azure, Cloudflare |
| **Payments & E-commerce** | 35 | Stripe, PayPal, Square, Razorpay, WeChat Pay, Shopify |
| **Notification & Collaboration** | 31 | Slack, Discord, Telegram, DingTalk, Feishu, Teams, Twilio |
| **Monitoring & Alerting** | 24 | Datadog, New Relic, Sentry, Grafana, PagerDuty |
| **Tokens** | 16 | JWT, Bearer, OAuth tokens, session tokens |
| **Databases & Connections** | 15 | MySQL, PostgreSQL, MongoDB, Redis, ES, InfluxDB |
| **Passwords** | 13 | Generic passwords, root/admin passwords, DB/protocol passwords |
| **Security Platforms** | 13 | Auth0, Shodan, Censys, VirusTotal, AbuseIPDB |
| **Private Keys & Certificates** | 11 | RSA, DSA, EC, OpenSSH, PKCS8 private keys and certificates |
| **Encoding & Fingerprints** | 8 | Hashes, UUIDs, long Base64 strings, SSH public keys |
| **Contact Information** | 3 | Phone numbers, email addresses, identity numbers |
| **Network Identifiers** | 3 | IPv4, private IPs, MAC addresses |
| **API Keys** | 3 | Generic API key / access key |
| **WeChat Ecosystem** | 3 | AppID, CorpID, Secret |
| **URL / API** | 2 | URLs, API endpoints |
| **Paths** | 1 | File paths, system paths |
| **Domains** | 1 | Domain names with TLD validation |

### False-positive Control Strategy

To keep the 920-rule ruleset usable, the scanner applies several filtering layers by default:

- **Blacklist filtering** - skips common static resource names, framework API fragments, and obviously non-sensitive content
- **Context filtering** - continues to enforce TLD validation, JS API context checks, and path-length checks
- **Placeholder filtering** - excludes examples like `your_api_key`, `replace_me`, `changeme`, and `placeholder`
- **Masked-value filtering** - excludes masked text such as `xxxxxx`, `******`, and `<token>`
- **Weak-value filtering** - adds minimum length, character-shape, and common-word checks for credential-like matches

### Scan and Export Behavior

- `-sensitive=true` generates `sensitive_report.xlsx` and `sensitive_report.html`
- `-postman=true` generates `api_collection.postman_collection.json`
- `-postman` is independent from `-sensitive`
- `scan-only` reuses the same scanner and JS deobfuscation pipeline
- If the HTTP method cannot be inferred reliably, Postman exports `UNKNOWN`
- Relative API paths remain unchanged and are not prefixed with any `baseUrl`

## 🧭 Page and Route Map

After unpacking, the output directory also includes:

- `route_manifest.json`: machine-readable page and route structure
- `route_map.md`: human-readable route description
- `route_map.mmd`: Mermaid graph for navigation rendering

Current route recovery covers:

- Entry-page detection
- Main-package and subpackage page lists
- TabBar pages
- Page titles and file mapping
- `usingComponents` dependencies
- `wx.navigateTo`, `redirectTo`, `reLaunch`, and `switchTab`
- Static WXML `<navigator>` navigation
- WXML `bindtap` / `catchtap` trigger-to-handler mapping
- `data-url`, `data-route`, `data-path`, and `data-page` backfilling
- Dynamic route hints from template strings, string concatenation, and `dataset.url`
- Trigger text, trigger event, and handler name association
- Direct API attribution from page scripts
- Indirect API attribution from shared service modules
- Cross-file call-chain recovery from page handler to local helper to shared helper to final navigation
- Shared router helper inventory for files such as `utils/router.js` and `common/nav.js`
- Orphan-page candidates where `Page(...)` exists but the page is missing from `app.json`

### Report Content

Generated Excel / HTML reports include:

- **Overview** - scan stats, risk distribution, and category summary
- **Category sheets** - built dynamically from actual matches, such as cloud, payment, collaboration, development, and SaaS
- **URL / API** - URL, API endpoint, and request context
- **Databases & Connections** - JDBC, MongoDB, Redis, ES, and other connection information
- **Password / Secret / Token** - passwords, generic secrets, and session or access tokens
- **WeChat Ecosystem** - WeChat-related configuration
- **Obfuscated Files** - file path, score, techniques, and restore state

Each finding includes:
- Content
- Occurrence count
- File path
- Line number
- Risk level

Obfuscated-file entries additionally include:
- Status (`restored` / `partial` / `flagged`)
- Score
- Techniques
- Tag (`[OBFUSCATED] ...`)

### Postman Collection Example

```json
{
  "info": {
    "name": "wx1234567890abcdef - API Collection"
  },
  "item": [
    {
      "name": "POST /api/user/login",
      "request": {
        "method": "POST",
        "url": {
          "raw": "/api/user/login"
        }
      }
    }
  ]
}
```

### Rule Configuration

- The built-in ruleset is used by default
- The program does not auto-write `config/rule.yaml`
- If you manually place `config/rule.yaml`, your custom rules override the built-in rules
- This is suitable when you want to adjust audit scope without modifying source code

---

## 📈 Performance Comparison (v2.7.3 vs v1.0)

| Metric | v1.0 | v2.7.3 | Improvement |
|--------|------|--------|-------------|
| **Scan speed** | Baseline | +50-70% | Regex precompilation |
| **False-positive control** | Basic regex-only scan | Multi-layer filtering | Blacklist + context + placeholder + weak-value filtering |
| **Data volume** | 127,185 items | ~3,000 items | Deduplication + filtering |
| **Output format** | JSON | Excel / HTML | Interactive reports |
| **Concurrency** | Fixed 10 workers | CPU*2 dynamic | Adaptive concurrency |
| **I/O performance** | Direct write | 256 KB buffer | Fewer system calls |

---

## 🔄 Version History

### v2.7.3 (2026-05-12) - Semantic and AST Deep Restoration

#### New
- Added AST-level variable/function renaming with default `-ast-rename=deep`, applying high/medium-confidence local variable, function parameter, and local function-name rewrites
- Added `off/report/safe/deep` AST modes, with runtime CLI warnings that explain the active strategy, rewrite scope, protected symbols, and rollback path
- Added `.gwxapkg/ast_rename_map.json`, `.gwxapkg/ast_rename_diff.md`, `.gwxapkg/ast_rename.patch`, and `.gwxapkg/pre_ast_sources` for traceable write-back and rollback
- Added API call-chain artifacts: `.gwxapkg/api_call_chain.json` and `.gwxapkg/api_call_chain.md`
- Added AST API audit pseudocode in `.gwxapkg/api_pseudo.md` and `.gwxapkg/pseudo_api/*.js`
- Added the `api-link` command to connect raw Burp requests from a file or stdin to source-level API entries

#### Improved
- `scan`, `all`, the default unpack command, and `semantic -dir` now support `-ast-rename`, `-ast-diff`, and `-ast-patch`
- AST write-back explicitly protects `exports.xxx`, object keys, strings, comments, WXML handlers, and global/public identifiers such as `wx/uni/getApp/require/module/exports`
- `semantic_module_map.json` now includes AST statistics, API call-chain paths, and API pseudocode paths for audit traceability

#### Verified
- Added regression tests for AST modes, confidence gating, diff/patch/rollback, API call chains, pseudocode, and Burp request linking
- Verified with `go test ./...` and `go build ./...`

### v2.7.2 (2026-04-30) - Stability and Report Fixes

#### Fixed
- Fixed a `nil pointer dereference` crash that could occur while deobfuscating JavaScript embedded in HTML `<script>` blocks during `all`
- Hardened `goja` AST traversal and variable-declaration detection against incomplete `VariableStatement`, `Binding`, and typed nil nodes
- Fixed static deobfuscation of `\x22`, `\u0022`, and similar escapes so decoded string content does not break JavaScript syntax
- Fixed `.xlsx` report generation when category names contain illegal Excel sheet characters
- Improved WXML restoration by unwrapping `virtual` nodes and trying safer fallback render arguments for dynamic branches
- If one JavaScript fragment fails deobfuscation analysis, Gwxapkg now keeps the original content and continues unpacking the package

#### Verified
- Added formatter, reporter, and unpack regression tests for JavaScript analysis panics, HTML embedded-script panics, incomplete AST nodes, Excel sheet-name sanitization, and WXML virtual-node unwrapping
- Verified with `go test ./...` and Windows amd64 cross-compilation

### v2.7.1 (2026-04-21) - Route Analysis Enhancements

#### New
- **Cross-page call-chain recovery** - writes page `handler -> local helper -> shared helper -> final navigation` chains into `route_manifest.json`
- **Shared router helper detection** - identifies shared navigation wrappers such as `utils/router.js` and `common/nav.js`, with page usage, methods, and target hints

#### Improved
- **Route report upgrades** - `route_map.md` and `route_map.mmd` now display call-chain and shared-helper information
- **Fallback edge pruning** - removes duplicate `UNKNOWN` fallback edges when a concrete navigation method has already been recovered

### v2.7.0 (2026-04-18) - Major Feature Upgrade

#### New
- **Interactive HTML report**
- **`scan-only` standalone mode**
- **GitHub Actions integration**
- **Windows AppName extraction**
- **Expanded batch processing**
- **Postman Collection export**
- **Default JS deobfuscation**
- **Obfuscated-file reporting**
- **Cache-path diagnostics**
- **Built-in rules by default**
- **Rule category normalization**
- **Stronger false-positive controls**

### v2.6.0 (2026-03-17) - Stable Release

#### New
- Stability improvements
- Ongoing rule optimization
- Better Windows / macOS / Linux compatibility
- Dependency updates

#### Fixed
- Subpackage handling issues in some cases
- Excel report issues with special characters
- Large Mini Program memory usage

### v2.5.0 (2025-12-05) - Major Update

#### New
- Multi-sheet Excel reporting
- Smarter false-positive filtering
- Deduplication
- Risk grading
- Full context with file paths and line numbers

#### Improved
- Dynamic concurrency
- Buffered I/O
- Rule precompilation
- Optimized build flags

#### Fixed
- Domain rules falsely matching file names
- JavaScript APIs falsely treated as domains
- Directory merge performance

### v1.0.0 (2024-XX-XX)
- Initial release
- Basic unpacking
- Code beautification
- Sensitive-data scanning with JSON output

---

## 🛠️ Technical Architecture

```text
Gwxapkg/
├── cmd/
│   └── root.go           # CLI entry, progress, report generation
├── internal/
│   ├── cmd/              # command handling and file parsing
│   ├── decrypt/          # AES + XOR decryption
│   ├── unpack/           # wxapkg binary parsing
│   ├── restore/          # project-structure restoration
│   ├── formatter/        # beautification and JS deobfuscation
│   │   ├── jsformatter.go
│   │   └── deobfuscator.go
│   ├── key/              # rule management and precompilation
│   ├── scanner/          # scanning engine
│   │   ├── types.go
│   │   ├── rule_meta.go
│   │   ├── filter.go
│   │   ├── collector.go
│   │   ├── scanner.go
│   │   └── api_extractor.go
│   ├── reporter/         # report generation
│   │   ├── excel.go
│   │   ├── html.go
│   │   └── postman.go
│   ├── config/           # configuration management
│   └── ui/               # terminal UI
├── config/
│   └── rule.yaml         # optional custom override file
└── main.go
```

---

## 🤝 Contributing

Contributions are welcome:

1. Fork this repository
2. Create a feature branch: `git checkout -b feature/AmazingFeature`
3. Commit your changes: `git commit -m 'Add some AmazingFeature'`
4. Push the branch: `git push origin feature/AmazingFeature`
5. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License. See [LICENSE](LICENSE).

---

## ❓ FAQ

### 1. Why does it “flash close” when double-clicked?
This is a command-line tool and should not be launched by double-clicking the executable.

- Wrong: double-clicking `gwxapkg.exe`
- Correct: open a terminal, `cd` into the tool directory, and run it from there

### 2. Why can’t it find my Mini Program package?
Make sure you have logged into desktop WeChat and opened the target Mini Program at least once. If it still cannot be found, run `scan` and verify whether the detected cache path is correct.

---

## 📩 Contact

If you add on WeChat, please include the reason. Basic questions such as how to open a terminal or install Go will not be answered.
