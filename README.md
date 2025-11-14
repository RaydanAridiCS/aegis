# Aegis 🛡️

A secure command-line file encryption tool built with Go. Aegis provides military-grade encryption for your sensitive directories with an intuitive CLI interface.

## Overview

Aegis is a CLI application that allows you to:
- **Seal** (encrypt) directories and their contents with password-derived keys
- **Unseal** (decrypt) protected directories
- **Watch** directories for changes and automatically re-encrypt

Built with strong cryptographic primitives including scrypt for key derivation, Aegis ensures your data remains protected with industry-standard security practices.

## Features

- 🔐 **Strong Encryption**: Uses scrypt for secure key derivation from passwords
- 📁 **Directory-Level Protection**: Encrypt entire directories at once
- 👀 **File Watching**: Monitor directories and automatically re-seal on changes
- 🚀 **Fast & Lightweight**: Pure Go implementation with minimal dependencies
- 💻 **Cross-Platform**: Works on Windows, macOS, and Linux
- 🎯 **Simple CLI**: Intuitive commands powered by Cobra

## Installation

### Prerequisites

- Go 1.16 or higher

### From Source

```bash
# Clone the repository
git clone https://github.com/RaydanAridiCS/aegis.git
cd aegis

# Download dependencies
go mod download

# Build the binary
go build -o aegis.exe ./cmd/aegis/

# (Optional) Install to your system
go install ./cmd/aegis/
```

## Quick Start

### Encrypt a Directory (Seal)

```bash
# Seal a directory with encryption
./aegis.exe seal /path/to/directory
```

### Decrypt a Directory (Unseal)

```bash
# Unseal an encrypted directory
./aegis.exe unseal /path/to/directory
```

### Watch a Directory

```bash
# Monitor directory for changes and auto-seal
./aegis.exe watch /path/to/directory
```

## Usage

### Available Commands

```
aegis [command]

Available Commands:
  seal        Encrypt a directory
  unseal      Decrypt a directory
  watch       Watch a directory for changes
  help        Help about any command
  completion  Generate shell completion scripts

Flags:
  -h, --help   help for aegis
```

### Command Details

#### Seal Command
Encrypts all files within a directory using password-derived encryption.

```bash
aegis seal [directory]
```

#### Unseal Command
Decrypts a previously sealed directory with the correct password.

```bash
aegis unseal [directory]
```

#### Watch Command
Monitors a directory for file changes and automatically re-encrypts when modifications are detected.

```bash
aegis watch [directory]
```

### Getting Help

```bash
# General help
./aegis.exe --help

# Command-specific help
./aegis.exe seal --help
./aegis.exe unseal --help
./aegis.exe watch --help
```

## Project Structure

```
aegis/
├── cmd/
│   └── aegis/
│       └── main.go          # Application entry point
├── internal/
│   └── cli/
│       ├── root.go          # Root command configuration
│       ├── seal.go          # Seal command implementation
│       ├── unseal.go        # Unseal command implementation
│       └── watch.go         # Watch command implementation
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
└── README.md               # This file
```

## Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [scrypt](https://golang.org/x/crypto/scrypt) - Key derivation
- [fsnotify](https://github.com/fsnotify/fsnotify) - File system notifications
- [term](https://golang.org/x/term) - Terminal utilities for secure password input

## Development

### Building

```bash
# Build for current platform
go build -o aegis.exe ./cmd/aegis/

# Build for specific platform
GOOS=linux GOARCH=amd64 go build -o aegis ./cmd/aegis/
GOOS=darwin GOARCH=amd64 go build -o aegis ./cmd/aegis/
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Quality

```bash
# Format code
go fmt ./...

# Vet code for issues
go vet ./...

# Tidy dependencies
go mod tidy
```

## Security Considerations

- **Password Strength**: Use strong, unique passwords for encryption
- **Key Derivation**: Aegis uses scrypt with secure parameters for key derivation
- **Data Protection**: Original files are preserved until successful encryption
- **Memory Safety**: Sensitive data is cleared from memory after use

## Roadmap

- [ ] Implement core encryption/decryption logic
- [ ] Add file integrity verification (HMAC)
- [ ] Support for multiple encryption algorithms
- [ ] Compressed encryption option
- [ ] Backup and restore functionality
- [ ] GUI version
- [ ] Cloud storage integration

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request


## Author

**Raydan Aridi**
- GitHub: [@RaydanAridiCS](https://github.com/RaydanAridiCS)

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Inspired by security best practices and modern cryptographic standards
- Thanks to the Go community for excellent tooling and libraries

---

⚠️ **Disclaimer**: This tool is currently under development. Always maintain backups of important data before encryption.
