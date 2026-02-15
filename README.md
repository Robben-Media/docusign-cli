# placeholder-cli

<!-- Replace with your CLI description -->

A CLI tool for [SERVICE_NAME] built with Go.

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap builtbyrobben/tap
brew install placeholder-cli
```

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/builtbyrobben/placeholder-cli/releases).

### Build from Source

```bash
git clone https://github.com/builtbyrobben/placeholder-cli.git
cd placeholder-cli
make build
```

## Authentication

### Set API Key

```bash
# Interactive (secure, recommended)
placeholder-cli auth set-key --stdin

# From environment variable
echo $API_KEY | placeholder-cli auth set-key --stdin

# From argument (discouraged - exposes in shell history)
placeholder-cli auth set-key YOUR_API_KEY
```

### Check Status

```bash
placeholder-cli auth status
```

### Remove Credentials

```bash
placeholder-cli auth remove
```

### Environment Variables

- `PLACEHOLDER_CLI_API_KEY` - Override stored credentials
- `PLACEHOLDER_CLI_KEYRING_BACKEND` - Force keyring backend (auto/keychain/file)
- `PLACEHOLDER_CLI_KEYRING_PASS` - Password for file backend (headless systems)

## Usage

<!-- Add your CLI usage examples here -->

```bash
placeholder-cli --help
```

## Development

### Prerequisites

- Go 1.22+
- Make

### Commands

```bash
make build        # Build binary
make test         # Run tests
make lint         # Run linter
make ci           # Run full CI suite
make tools        # Install dev tools
```

## License

MIT

## Contributing

Contributions are welcome! Please read our contributing guidelines before submitting PRs.
