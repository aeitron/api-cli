# Aeitron Command Line Interface (CLI)

The official command-line interface for managing Aeitron intelligent edge services, network routing, DNS, compute placement, security, and infrastructure.

## Installation

### Via Go

```bash
go install github.com/aeitron/api-cli/cmd@latest
```

Ensure `$GOPATH/bin` or `$HOME/go/bin` is in your system `PATH`.

### From Source

```bash
git clone https://github.com/aeitron/api-cli.git
cd api-cli
go build -o aeitron ./cmd
```

## Configuration

Set your Aeitron API token and target environment via environment variables:

```bash
export AEITRON_API_KEY="atk_live_your_api_key_here"
export AEITRON_ENDPOINT="https://api.aeitron.com"
```

Or configure via the CLI configuration command:

```bash
aeitron config set api_key atk_live_your_api_key_here
aeitron config set endpoint https://api.aeitron.com
```

## Commands Overview

The CLI provides subcommands across all infrastructure domains:

- `aeitron auth`: Manage authentication, API credentials, and sessions.
- `aeitron edge`: Manage edge proxying, caching, WAF rules, and routing policies.
- `aeitron network`: Query global routing topologies, PoP status, and latency metrics.
- `aeitron dns`: Manage authoritative DNS zones and records.
- `aeitron compute`: Manage workloads, microVM deployments, and placement targets.
- `aeitron security`: Audit security postures, inspect firewall rules, and rotate credentials.
- `aeitron billing`: View usage telemetry, cost allocations, and invoices.
- `aeitron events`: Subscribe to and inspect real-time CloudEvents audit trails.

## Usage Examples

### Inspect PoP Health

```bash
aeitron network pops list
```

### Manage DNS Records

```bash
# List records for a domain
aeitron dns list example.com

# Create an A record
aeitron dns create example.com --type A --name app --value 192.0.2.1 --ttl 300
```

### Edge Cache Purge

```bash
aeitron edge cache purge example.com --all
```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
