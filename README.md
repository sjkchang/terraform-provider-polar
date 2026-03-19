# Terraform Provider for Polar

The Polar provider enables [Terraform](https://www.terraform.io) to manage [Polar.sh](https://polar.sh) resources such as organizations, products, meters, benefits, and webhook endpoints.

> **Warning:** This provider depends on the [Polar Go SDK](https://github.com/polarsource/polar-go), which is still pre-1.0 and frequently receives breaking changes to its API. As a result, this Terraform provider may also introduce breaking changes between minor versions until the underlying SDK and API stabilize.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.25 (to build the provider)

## Installation

The provider is available on the [Terraform Registry](https://registry.terraform.io/providers/sjkchang/polar/latest). Terraform will automatically download it when you run `terraform init`:

```hcl
terraform {
  required_providers {
    polar = {
      source  = "sjkchang/polar"
      version = "~> 0.1"
    }
  }
}
```

## Authentication

The provider requires a Polar organization access token. You can provide it via the `access_token` attribute or the `POLAR_ACCESS_TOKEN` environment variable:

```hcl
provider "polar" {
  # Reads from POLAR_ACCESS_TOKEN env var by default
  # access_token = "polar_oat_xxx"

  # Required: must be "production" or "sandbox".
  # server = "production"
}
```

## Resources

- **polar_organization** — Adopt and configure organization settings (profile, subscriptions, notifications, feature flags)
- **polar_product** — Manage products with fixed, free, custom, and metered pricing
- **polar_meter** — Track usage events with configurable filters and aggregations
- **polar_benefit** — Define benefits like custom perks, license keys, meter credits, Discord roles, GitHub repo access, and downloadables
- **polar_webhook_endpoint** — Configure webhook endpoints for event notifications

## Data Sources

- **polar_meter** — Fetch an existing meter by ID
- **polar_benefit** — Fetch an existing benefit by ID

## Example Usage

```hcl
# Create a meter to track API calls
resource "polar_meter" "api_calls" {
  name = "API Calls"

  filter = {
    conjunction = "and"
    clauses = [{
      property = "name"
      operator = "eq"
      value    = "api_call"
    }]
  }

  aggregation = {
    func = "count"
  }
}

# Create a monthly subscription product with metered pricing
resource "polar_product" "api_access" {
  name               = "API Access"
  description        = "Pay per API call."
  recurring_interval = "month"

  prices = [{
    amount_type = "metered_unit"
    meter_id    = polar_meter.api_calls.id
    unit_amount = "0.01"
    cap_amount  = 50000
  }]
}

# Set up a webhook to receive order notifications
resource "polar_webhook_endpoint" "orders" {
  url    = "https://example.com/webhooks/polar"
  format = "raw"
  events = ["order.created", "order.paid"]
}
```

See the [examples](examples/) directory and [provider documentation](docs/) for more.

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
