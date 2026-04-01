# Cloudflare Plugin for formae

A formae resource plugin for managing Cloudflare infrastructure, including DNS zones, DNS records, and email routing.

## Installation

```bash
# Build and install the plugin
make install
```

## Supported Resources

| Resource Type | Description |
|---------------|-------------|
| `Cloudflare::DNS::Zone` | DNS zone (domain) configuration |
| `Cloudflare::DNS::Record` | DNS records (A, AAAA, CNAME, MX, TXT, etc.) |
| `Cloudflare::Email::RoutingRule` | Email routing rules |
| `Cloudflare::Email::DestinationAddress` | Verified destination email addresses |

## Configuration

### Environment Variables

Set the following environment variable for authentication:

```bash
export CLOUDFLARE_API_TOKEN="your-api-token"
```

You can create an API token in the [Cloudflare Dashboard](https://dash.cloudflare.com/profile/api-tokens).

### Target Configuration

Configure a target in your Forma file:

```pkl
import "@formae/formae.pkl"

new formae.Target {
    label = "cloudflare"
    namespace = "CLOUDFLARE"
    config = new Mapping {
        ["AccountId"] = "your-account-id"
        ["ZoneId"] = "your-zone-id"  // Optional, for zone-scoped operations
    }
}
```

## Examples

### DNS Zone and Records

```pkl
import "@cloudflare/dns/zone.pkl" as dnsZone
import "@cloudflare/dns/record.pkl" as dnsRecord

// Create a DNS zone
new dnsZone.Zone {
    label = "my-zone"
    name = "example.com"
    account_id = "your-account-id"
    `type` = "full"
}

// Create an A record
new dnsRecord.Record {
    label = "www-record"
    zone_id = "your-zone-id"
    name = "www"
    `type` = "A"
    content = "192.0.2.1"
    ttl = 300
    proxied = true
}
```

### Email Routing

```pkl
import "@cloudflare/email/routing_rule.pkl" as emailRule
import "@cloudflare/email/destination_address.pkl" as emailDest

// Register a destination address (must be verified in Cloudflare dashboard)
new emailDest.DestinationAddress {
    label = "my-email"
    account_id = "your-account-id"
    email = "me@gmail.com"
}

// Create a catch-all forwarding rule
new emailRule.RoutingRule {
    label = "catch-all"
    zone_id = "your-zone-id"
    name = "Forward all emails"
    enabled = true

    matchers = new Listing {
        new emailRule.Matcher { `type` = "all" }
    }

    actions = new Listing {
        new emailRule.Action {
            `type` = "forward"
            value = new Listing { "me@gmail.com" }
        }
    }
}
```

See the [examples/](examples/) directory for complete usage examples:

- `examples/basic/` - DNS zone and records
- `examples/email-routing/` - Email routing configuration

```bash
# Evaluate an example
formae eval examples/basic/main.pkl

# Apply resources
formae apply --mode reconcile --watch examples/basic/main.pkl
```

## Development

### Prerequisites

- Go 1.25+
- [Pkl CLI](https://pkl-lang.org/main/current/pkl-cli/index.html)
- Cloudflare API token (for conformance testing)

### Building

```bash
make build      # Build plugin binary
make test       # Run unit tests
make lint       # Run linter
make install    # Build + install locally
```

### Local Testing

```bash
# Install plugin locally
make install

# Start formae agent
formae agent start

# Apply example resources
formae apply --mode reconcile --watch examples/basic/main.pkl
```

### Conformance Testing

Conformance tests validate the plugin's CRUD lifecycle using test fixtures in `testdata/`.

Required environment variables:
- `CLOUDFLARE_API_TOKEN` - Your Cloudflare API token
- `CLOUDFLARE_ZONE_ID` - A test zone ID to create records in

| File | Purpose |
|------|---------|
| `resource.pkl` | Initial resource creation |
| `resource-update.pkl` | In-place update (mutable fields) |
| `resource-replace.pkl` | Replacement (createOnly fields) |

```bash
make conformance-test                  # Latest formae version
make conformance-test VERSION=0.81.0   # Specific version
```

The `scripts/ci/clean-environment.sh` script cleans up test resources.

## Supported DNS Record Types

| Type | Description |
|------|-------------|
| `A` | IPv4 address |
| `AAAA` | IPv6 address |
| `CNAME` | Canonical name (alias) |
| `MX` | Mail exchange |
| `TXT` | Text record |
| `NS` | Name server |
| `SRV` | Service locator |
| `CAA` | Certificate Authority Authorization |
| `PTR` | Pointer record |
| `HTTPS` | HTTPS service binding |
| `SVCB` | Service binding |

## Licensing

This plugin is licensed under the Apache 2.0 license.

See the formae plugin policy: <https://docs.formae.io/plugin-sdk/>
