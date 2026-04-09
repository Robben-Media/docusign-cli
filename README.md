# docusign-cli

A CLI tool for the [DocuSign eSignature API](https://developers.docusign.com/) built with Go. Manage envelopes, documents, recipients, templates, and embedded signing from the command line.

## Installation

### Download Binary

Download the latest release from [GitHub Releases](https://github.com/builtbyrobben/docusign-cli/releases).

### Build from Source

```bash
git clone https://github.com/builtbyrobben/docusign-cli.git
cd docusign-cli
make build
```

## Configuration

docusign-cli uses OAuth 2.0 to authenticate with the DocuSign API. You need a DocuSign integration key (client ID) and secret key from the [DocuSign Admin Console](https://admindemo.docusign.com/).

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DOCUSIGN_INTEGRATION_KEY` | Integration key (client ID) |
| `DOCUSIGN_SECRET_KEY` | Secret key |

### Initial Setup

```bash
# 1. Store your integration key and secret
docusign-cli auth set-credentials

# 2. Authenticate via OAuth 2.0 browser flow
docusign-cli auth login

# Check authentication status
docusign-cli auth status

# Remove all credentials and tokens
docusign-cli auth remove
```

Tokens are stored locally and auto-refreshed when expired.

## Commands

### auth -- Authentication and credentials

```bash
docusign-cli auth set-credentials    # Set integration key and secret key
docusign-cli auth login              # OAuth 2.0 login flow
docusign-cli auth status             # Show authentication status
docusign-cli auth remove             # Remove all credentials and tokens
```

### envelopes -- Envelope operations

```bash
# List envelopes
docusign-cli envelopes list

# Filter by status and date
docusign-cli envelopes list --status sent --from 2026-01-01 --count 25

# Get envelope details
docusign-cli envelopes get <envelope-id>

# Create and send an envelope with a document
docusign-cli envelopes create \
  --subject "Please sign this contract" \
  --signer-email jane@example.com \
  --signer-name "Jane Doe" \
  --document ./contract.pdf

# Create as draft (without sending)
docusign-cli envelopes create \
  --subject "Draft agreement" \
  --signer-email jane@example.com \
  --signer-name "Jane Doe" \
  --document ./agreement.pdf \
  --status created

# Send a draft envelope
docusign-cli envelopes send <envelope-id>

# Void an envelope
docusign-cli envelopes void <envelope-id> --reason "Sent in error"

# Get envelope audit trail
docusign-cli envelopes audit <envelope-id>
```

### documents -- Document operations

```bash
# List documents in an envelope
docusign-cli documents list <envelope-id>

# Download a document as PDF
docusign-cli documents download <envelope-id> <document-id> --output ./signed.pdf

# Download to stdout
docusign-cli documents download <envelope-id> <document-id> > signed.pdf
```

### recipients -- Recipient operations

```bash
# List recipients of an envelope
docusign-cli recipients list <envelope-id>
```

### templates -- Template operations

```bash
# List templates
docusign-cli templates list

# Search templates
docusign-cli templates list --search "NDA"

# Limit results
docusign-cli templates list --count 25

# Get template details
docusign-cli templates get <template-id>
```

### views -- Embedded signing

```bash
# Create an embedded signing URL
docusign-cli views signing <envelope-id> \
  --signer-email jane@example.com \
  --signer-name "Jane Doe" \
  --return-url https://example.com/done \
  --client-user-id 1001
```

### version

```bash
docusign-cli version
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output JSON to stdout (for scripting) |
| `--plain` | Output stable TSV text (no colors) |
| `--verbose` | Enable verbose logging |
| `--force` | Skip confirmation prompts |
| `--no-input` | Never prompt; fail instead (CI mode) |
| `--color` | Color output: `auto`, `always`, or `never` |

## License

MIT
