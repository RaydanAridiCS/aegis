# Aegis 🛡️

A secure command-line file encryption tool built with Go. Aegis provides military-grade encryption for your sensitive directories with an intuitive CLI interface.

## Overview

Aegis is a CLI application that allows you to:
- **Seal** (encrypt) directories and their contents with password-derived keys
- **Unseal** (decrypt) protected directories
- **Watch** directories for changes with detailed logging and diff tracking

Built with strong cryptographic primitives including AES-256-GCM and scrypt for key derivation, Aegis ensures your data remains protected with industry-standard security practices.

## Features

- 🔐 **AES-256-GCM Encryption**: Military-grade authenticated encryption with scrypt key derivation
- 📁 **Directory-Level Protection**: Encrypt entire directories recursively at once
- 🔑 **Secure Key Derivation**: Uses scrypt (N=32768, r=8, p=1) for password-based key generation
- 👀 **Advanced File Watching**: Monitor directories with detailed change detection and diff logging
- 📝 **Dual Logging System**: Generates both detailed and basic log files for watch sessions
- 🔒 **Extension Preservation**: Original file extensions are encrypted and restored on decryption
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
# Monitor directory for changes and log all modifications
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

Monitors a directory for file changes and logs all modifications with detailed diff information. Creates both detailed and basic log files in a `logs/` directory.

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
- **Key Derivation**: Aegis uses scrypt with secure parameters (N=32768, r=8, p=1) for key derivation
- **Authenticated Encryption**: AES-256-GCM provides both confidentiality and integrity
- **Unique Cryptographic Material**: Each file gets a unique salt and nonce
- **Extension Protection**: Original file extensions are embedded in encrypted data
- **Memory Safety**: Sensitive data is cleared from memory after use

## Technical Details

### Encryption Process (Seal)

1. Generate a unique 16-byte salt per file
2. Derive a 256-bit key using scrypt (password + salt)
3. Create AES-256-GCM cipher
4. Generate a unique nonce for each encryption
5. Embed original file extension in plaintext
6. Encrypt and authenticate data
7. Output format: `[Salt][Nonce][Ciphertext+AuthTag]`

### Decryption Process (Unseal)

1. Extract salt from encrypted file header
2. Re-derive key using scrypt (password + stored salt)
3. Extract nonce and ciphertext
4. Decrypt and verify authentication tag
5. Recover original file extension
6. Restore file with original name and extension

## Author

**Raydan Aridi**
- GitHub: [@RaydanAridiCS](https://github.com/RaydanAridiCS)

## Collaborators

Current collaborators
- Raydan Aridi — Maintainer — https://github.com/RaydanAridiCS
- Waseem Bou Hamdan - Collaboratotr - https://github.com/WaseemBouHamdan
- Hasan Farhat - Collaborator - 

## Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) CLI framework
- Inspired by security best practices and modern cryptographic standards
- Thanks to the Go community for excellent tooling and libraries

---

⚠️ **Disclaimer**: Always maintain backups of important data before encryption.
