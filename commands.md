Here are all the commands you'll need when cloning the Aegis repository to a new device:

## Initial Setup

```powershell
# Clone the repository
git clone https://github.com/RaydanAridiCS/aegis.git
cd aegis

# Download all dependencies (restores from go.mod)
go mod download

# Verify dependencies are installed
go mod verify
```

## Build Commands

```powershell
# Build the project (creates aegis.exe in current directory)
go build -o aegis.exe ./cmd/aegis/

# Or build without specifying output name (creates aegis.exe automatically on Windows)
go build ./cmd/aegis/

# Build and install to $GOPATH/bin (makes 'aegis' available system-wide)
go install ./cmd/aegis/
```

## Testing Commands

```powershell
# Run all tests in the project
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with coverage report
go test -cover ./...

# Run tests with detailed coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run tests for a specific package
go test ./internal/cli/
```

## Running the Application

```powershell
# Run without building (good for development)
go run ./cmd/aegis/

# Run the compiled binary - show help
./aegis.exe

# Test seal command
./aegis.exe seal ./test-directory

# Test unseal command
./aegis.exe unseal ./test-directory

# Test watch command
./aegis.exe watch ./test-directory

# Get help for specific commands
./aegis.exe seal --help
./aegis.exe unseal --help
./aegis.exe watch --help
```

## Development Commands

```powershell
# Format all Go files
go fmt ./...

# Check for common issues
go vet ./...

# Tidy up dependencies (removes unused, adds missing)
go mod tidy

# Update dependencies to latest versions
go get -u ./...
go mod tidy
```

## Complete Setup & Test Workflow

```powershell
# 1. Clone and enter directory
git clone https://github.com/RaydanAridiCS/aegis.git
cd aegis

# 2. Download dependencies
go mod download

# 3. Build the project
go build -o aegis.exe ./cmd/aegis/

# 4. Test the binary
./aegis.exe
./aegis.exe seal test-dir
./aegis.exe unseal test-dir
./aegis.exe watch test-dir

# 5. Run any tests (when you create them)
go test -v ./...
```

**Note:** Make sure Go 1.16+ is installed on the new device before running these commands.