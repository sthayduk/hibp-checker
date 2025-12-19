# HIBP Checker

A fast, concurrent command-line tool written in Go to check NTLM password hashes against the [Have I Been Pwned](https://haveibeenpwned.com/) Pwned Passwords API.

## Features

- **Concurrent API Queries**: Configurable worker pool for parallel API requests
- **Memory Efficient**: Streams results to output file as they're discovered
- **Smart Filtering**: Automatically skips computer accounts and empty hashes
- **Interrupt Safe**: Results are written immediately, preserving data if the process is interrupted
- **Real-time Progress**: Live console output showing query progress and elapsed time
- **Flexible Input**: Supports custom delimiters and header row skipping

## Installation

### Prerequisites

- Go 1.21 or later

### Build from Source

```bash
git clone https://github.com/itdesign/hibp-checker-go.git
cd hibp-checker-go
go build -o hibp-checker
```

### Cross-Platform Build

Use the included build script to compile for all supported platforms:

```bash
./build.sh
```

This creates binaries in the `dist/` directory:

| Platform | Binary |
| --- | --- |
| Windows x64 | `hibp-checker-windows-amd64.exe` |
| Linux x64 | `hibp-checker-linux-amd64` |
| macOS Intel | `hibp-checker-darwin-amd64` |
| macOS Apple Silicon | `hibp-checker-darwin-arm64` |

## Usage

### Basic Usage

```bash
./hibp-checker check -i accounts.txt -o exposed.txt
```

### Command Line Options

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--input` | `-i` | string | (required) | Input file containing account:hash pairs |
| `--output` | `-o` | string | - | Output file for exposed accounts |
| `--delimiter` | `-d` | string | `:` | Delimiter between account and hash |
| `--skip-header` | `-s` | bool | `false` | Skip first line (header row) |
| `--workers` | `-w` | int | `10` | Number of concurrent workers |
| `--limit` | `-l` | int | `0` | Limit accounts to check (0 = unlimited) |

### Examples

**Check accounts with default settings:**
```bash
./hibp-checker check -i ntlm_hashes.txt -o exposed.txt
```

**Use 20 concurrent workers for faster processing:**
```bash
./hibp-checker check -i ntlm_hashes.txt -o exposed.txt -w 20
```

**CSV file with semicolon delimiter and header row:**
```bash
./hibp-checker check -i accounts.csv -o exposed.txt -d ";" -s
```

**Check only first 100 accounts (for testing):**
```bash
./hibp-checker check -i ntlm_hashes.txt -o exposed.txt -l 100
```

## Input File Format

The input file should contain one account per line in the format:

```
account:hash
```

**Example:**
```
john.doe:8846F7EAEE8FB117AD06BDD830B7586C
jane.smith:5F4DCC3B5AA765D61D8327DEB882CF99
admin:A4F49C406510BDCAB6824EE7C30FD852
```

### Notes on Input

- Computer accounts (ending with `$`) are automatically skipped
- Empty hashes are ignored
- Hash comparison is case-insensitive
- Lines with invalid format are skipped

## Output

### Console Output

```
Starting HIBP check for 1500 accounts (1423 unique prefixes)...
[EXPOSED] john.doe
[EXPOSED] jane.smith
Progress: 500/1423 prefixes queried (35.14%) - Elapsed: 45s
Progress: 1000/1423 prefixes queried (70.27%) - Elapsed: 92s
[EXPOSED] admin

Check complete. 3 exposed accounts found.
```

### Output File

The output file contains one exposed account per line:

```
john.doe
jane.smith
admin
```

## How It Works

1. **Load Accounts**: Reads account:hash pairs from the input file
2. **Build Prefix Index**: Groups accounts by their hash prefix (first 5 characters)
3. **Query HIBP API**: Concurrently queries the HIBP API for each unique prefix
4. **Match Results**: Compares returned hash suffixes against loaded hashes
5. **Stream Results**: Writes exposed accounts to output file immediately upon detection

### HIBP API Integration

This tool uses the [HIBP Pwned Passwords Range API](https://haveibeenpwned.com/API/v3#SearchingPwnedPasswordsByRange) which implements k-anonymity:

- Only the first 5 characters of the hash are sent to the API
- The API returns all hash suffixes matching that prefix
- Full hash comparison happens locally, preserving privacy

## Architecture

```
hibp-checker-go/
├── main.go                 # Entry point
├── cmd/
│   ├── root.go            # Root command setup
│   └── check.go           # Check command implementation
└── internal/
    └── hibp/
        ├── client.go      # HIBP API client
        └── checker.go     # Core checking logic
```

## Performance Considerations

- **Worker Count**: Default is 10 workers. Increase for faster processing, but be mindful of rate limits
- **Memory Usage**: The tool processes results inline without storing all API responses
- **Network**: Each unique hash prefix requires one API request

### Typical Performance

- **Benchmark**: 236,364 prefixes queried in ~10 minutes using 20 workers (~390 queries/second)
- Actual speed depends on network latency and HIBP API response times

## License

MIT License

## Acknowledgments

- [Have I Been Pwned](https://haveibeenpwned.com/) by Troy Hunt for providing the Pwned Passwords API
- [Cobra](https://github.com/spf13/cobra) for the CLI framework
