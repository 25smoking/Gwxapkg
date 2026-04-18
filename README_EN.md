# Gwxapkg

<div align="center">

![Version](https://img.shields.io/badge/version-2.6.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey.svg)
![Build](https://img.shields.io/badge/build-passing-brightgreen.svg)

**[中文](README.md) | [English](README_EN.md) | [日本語](README_JA.md)**

A powerful WeChat Mini Program `.wxapkg` unpacking tool built with Go, featuring automatic scanning, decryption, decompilation, and security analysis.

</div>

---

## ✨ Key Features

### 🔍 Smart Unpacking
- **Auto Scan** - Automatically detect macOS/Windows WeChat Mini Program cache directories
- **Auto Decrypt** - Support encrypted wxapkg file decryption (PC version)
- **One-Click Process** - Automatically find and process all files for specified AppID
- **Subpackage Handling** - Correctly handle main package and subpackage dependencies

### 🎨 Code Restoration
- **Complete Restoration** - Full support for wxml/wxss/js/json/wxs  
- **Code Beautification** - Auto-format JavaScript/CSS/HTML code
- **Directory Structure** - Restore original Mini Program project structure
- **Resource Extraction** - Complete extraction of images/audio/video resources

### 🔒 Security Analysis ⭐ NEW
- **Smart Scanning** - 200+ sensitive information detection rules
- **False Positive Filtering** - Intelligent blacklist, reduces false positives from 95% to 10-15%
- **Data Deduplication** - Auto-remove duplicate data for precision
- **Excel Reports** - Professional multi-sheet classified reports with file paths and line numbers
- **Risk Classification** - Automatic high/medium/low risk categorization

### ⚡ Performance Optimization
- **Dynamic Concurrency** - Auto-adjust concurrency based on CPU cores
- **Buffered I/O** - 256KB buffer for significantly improved file read/write performance
- **Rule Precompilation** - Compile regex at startup to avoid repeated overhead  
- **Build Optimization** - Use optimized build flags to reduce size and improve speed

---

## 📊 Supported File Types

| File Type | Support | Description |
|-----------|---------|-------------|
| `.wxml` | ✅ | Page structure restoration |
| `.wxss` | ✅ | Style file restoration |
| `.js` | ✅ | JavaScript code restoration + beautification |
| `.json` | ✅ | Configuration file extraction |
| `.wxs` | ✅ | WXS script restoration |
| Images/Audio/Video | ✅ | Complete resource file extraction |

---

## 📥 Installation

### Option 1: Download Precompiled Binary (Recommended)

Visit the [Releases](https://github.com/25smoking/Gwxapkg/releases) page to download the executable for your platform.

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/25smoking/Gwxapkg.git
cd Gwxapkg

# Build (optimized version)
go build -ldflags="-s -w" -o gwxapkg .

# Or run directly
go run . -h
```

**Requirements:** Go 1.21 or higher

---

## 🚀 Quick Start

### Basic Usage

```bash
# Auto scan and process Mini Program by AppID
./gwxapkg all -id=<AppID>

# List all available Mini Programs
./gwxapkg scan

# Unpack single wxapkg file
./gwxapkg -id=<AppID> -in=<file_path>

# Repack
./gwxapkg repack -in=<directory_path>
```

### Command Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `-id` | Mini Program AppID (required) | - |
| `-in` | Input file/directory path | - |
| `-out` | Output directory | auto-generated |
| `-restore` | Restore project directory structure | true |
| `-pretty` | Beautify code output | true |
| `-sensitive` | Enable sensitive information scanning | true |
| `-noClean` | Keep intermediate temporary files | false |
| `-save` | Save decrypted files | false |

### Usage Examples

```bash
# Example 1: Auto scan and unpack
./gwxapkg all -id=wx3c19e32cb8f31289

# Example 2: Unpack single file only
./gwxapkg -id=wx123456 -in=test.wxapkg -out=./output

# Example 3: Repack
./gwxapkg repack -in=./source_dir -out=new.wxapkg
```

---

## 📁 WeChat Mini Program Cache Locations

### macOS
```
~/Library/Containers/com.tencent.xinWeChat/Data/Library/Caches/
├── applet/
│   ├── release/
│   └── debug/
└── ...
```

### Windows
```
%USERPROFILE%\Documents\WeChat Files\Applet\
├── wx<appid>/
│   ├── <version>/
│   │   ├── __APP__.wxapkg      # Main package
│   │   └── __SUBCONTEXT__.wxapkg  # Subpackage
│   └── ...
└── ...
```

---

## 🎯 Sensitive Information Scanning

### Scanning Rules (200+)

| Category | Rules | Examples |
|----------|-------|----------|
| **Paths** | 1 | File paths, system paths |
| **URLs** | 2 | HTTP/HTTPS links, API endpoints |
| **Domains** | 1 | Domain addresses (TLD validation) |
| **Passwords** | 12+ | Various passwords, database credentials |
| **API Keys** | 40+ | AWS/Alibaba Cloud/Tencent Cloud keys |
| **Tokens** | 30+ | JWT/Bearer/OAuth tokens |
| **Database** | 15+ | MySQL/MongoDB/Redis connection strings |
| **Contact Info** | 3 | Phone/Email/ID numbers |
| **WeChat** | 4 | AppID/Secret/Webhook |  
| **Others** | 90+ | Certificates/Hashes/UUIDs etc. |

### Excel Report Contents

Generated reports include the following sheets:

- **Overview** - Scan statistics, risk distribution, category summary
- **Paths** - All path-related sensitive information
- **URLs** - All URLs and API endpoints
- **Domains** - Domain addresses (false positives filtered)
- **Passwords** - Password and credential information
- **API Keys** - Various cloud service keys
- **Tokens** - Access tokens and session information
- **Database** - Database connection information
- **Contact Info** - Phone numbers, emails, etc.
- **WeChat** - WeChat-related configurations
- **Others** - Other sensitive information

Each entry contains:
- ✅ Content
- ✅ Occurrence count
- ✅ File path  
- ✅ Line number
- ✅ Risk level

---

## 📈 Performance Comparison (v2.5.0 vs v1.0)

| Metric | v1.0 | v2.5.0 | Improvement |
|--------|------|--------|-------------|
| **Scan Speed** | Baseline | +50-70% | ⬆️⬆️⬆️ Rule precompilation |
| **False Positive Rate** | ~95% | 10-15% | ⬇️⬇️⬇️ Smart filtering |
| **Data Volume** | 127,185 items | ~3,000 items | ⬇️⬇️⬇️ Dedup + filtering |
| **Output Format** | JSON | Excel | ✅ Professional reports |
| **Concurrency** | Fixed 10 | Dynamic CPU*2 | ⬆️⬆️ Adaptive |
| **I/O Performance** | Direct write | 256KB buffer | ⬆️⬆️ Fewer syscalls |

---

## 🔄 Version History

### v2.5.0 (2025-12-05) - 🎉 Major Update

#### 🆕 New Features
- ✨ **Excel Report Generation** - Professional multi-sheet classified reports replacing simple JSON
- 🎯 **Smart False Positive Filtering** - Blacklist + TLD validation + context detection, 85% reduction in false positives
- 📊 **Data Deduplication** - Auto-deduplication, 97% reduction in data volume
- 🏷️ **Risk Classification** - Automatic high/medium/low risk categorization
- 📍 **Complete Context** - Each entry includes file path and line number

#### ⚡ Performance Optimizations
- 🚀 **Dynamic Concurrency** - Auto-adjust worker count based on CPU cores (previously fixed 10 → CPU*2)
- 💾 **Buffered I/O** - 256KB buffer improves file read/write performance
- 🔧 **Rule Precompilation** - Compile all regex at startup to avoid repeated overhead
- 📦 **Build Optimization** - Use `-ldflags="-s -w"` to reduce binary size

#### 🐛 Bug Fixes
- Fixed domain rule falsely matching filenames (e.g., index.weapp)
- Fixed JavaScript APIs being misidentified as domains
- Optimized directory merge performance

#### 💡 Technical Improvements
- Added `internal/scanner` module (types, filter, collector, scanner)
- Added `internal/reporter` module (Excel report generation)
- Use `excelize/v2` library to generate professional Excel reports
- Complete unit test coverage

### v1.0.0 (2024-XX-XX)
- 🎉 Initial release
- ✅ Basic unpacking functionality
- ✅ Code beautification
- ✅ Sensitive information scanning (JSON output)

---

## 🛠️ Technical Architecture

```
Gwxapkg/
├── cmd/
│   └── root.go           # CLI entry, progress bar, report generation
├── internal/
│   ├── cmd/              # Command processing, file parsing
│   ├── decrypt/          # AES+XOR decryption
│   ├── unpack/           # wxapkg binary parsing
│   ├── restore/          # Project structure restoration
│   ├── formatter/        # Code beautification (JS/CSS/HTML)
│   ├── key/              # Rule management, precompilation
│   ├── scanner/          # ⭐ NEW Scanning engine
│   │   ├── types.go      # Data models
│   │   ├── filter.go     # False positive filtering
│   │   ├── collector.go  # Data collection and deduplication
│   │   └── scanner.go    # Scanning logic
│   ├── reporter/         # ⭐ NEW Report generation
│   │   └── excel.go      # Excel reports
│   ├── config/           # Configuration management
│   └── ui/               # Terminal UI
├── config/
│   └── rule.yaml         # 200+ sensitive info rules
└── main.go
```

---

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork this repository
2. Create feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to branch (`git push origin feature/AmazingFeature`)
5. Create Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

---

## ❓ FAQ

### 1. Why does it "crash" or close immediately when double-clicked?
**This is a Command Line Interface (CLI) tool** and cannot be run by double-clicking.
- **Wrong way**: Double-clicking `gwxapkg.exe` in File Explorer. This causes the window to close immediately after the program finishes or errors out.
- **Correct way**: Open a terminal (CMD, PowerShell, or Terminal), `cd` to the tool's directory, and run the command there.

### 2. Mini Program package not found?
Ensure you have logged into the PC version of WeChat and opened the target Mini Program. If it still cannot be found, try using the `scan` command to manually verify if the detected path is correct.

---

## 📩 Contact

Please specify your purpose when adding on WeChat. **Note: Basic "1+1" level questions (e.g., how to open a terminal, how to install Go, etc.) will NOT be answered. Please use a search engine.**

<img src="https://i.imgur.com/9PxS5IK.jpeg" width="300" />

---

## ⚠️ Disclaimer

This tool is for educational and research purposes only. Do not use for illegal purposes. Users are responsible for any consequences resulting from using this tool.

---

## 🌟 Star History

If this project helps you, please give it a ⭐ Star!

---

<div align="center">

**Made with ❤️ by [25smoking](https://github.com/25smoking)**

</div>
