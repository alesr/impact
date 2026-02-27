# IMPACT

IMPACT is an *experimental* CLI for assessing the environmental impact of Scaleway infrastructure from two sources: Terraform plans (forecast) and Scaleway APIs (measured data).

It reports monthly impact focused on carbon emissions (`kgCO2e`) and water consumption (`m3`).

![Impact estimate](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/alesr/impact/main/badges/impact-estimate.json)

Estimated monthly impact badge (from Terraform plan). Setup and generation details: `docs/badges.md`.

> ⚠️ Experimental tool under development  
> This project is unofficial and not affiliated with or endorsed by Scaleway.  
> impact plan relies on Terraform-to-SKU mapping logic and catalog metadata, so results are estimates only.  
> Do not use this output as a compliance source of truth.  

## Commands

- `impact plan` - estimate impact from Terraform plans
- `impact actual` - query measured footprint from Scaleway APIs
- `impact doctor` - check environment/auth and API reachability
- `impact completion` - generate shell completions

Use help at any level:

```bash
impact --help
impact plan --help
impact actual --help
impact doctor --help
```

## Screenshots

### 1) CLI output

![CLI output](https://github.com/user-attachments/assets/d91aa997-0a5f-475c-bd4d-ae76521211b5)

### 2) TUI output

![TUI output](https://github.com/user-attachments/assets/a6660a24-3c87-450a-aa0c-1d53397dc7f6)

## What It Is For

- review infrastructure changes before `terraform apply`
- compare estimated (`plan`) vs measured (`actual`) impact
- export table or JSON output for reviews and reporting

## Installation

Build locally:

```bash
go build -o impact ./cmd/impact
```

Install from GitHub:

```bash
go install github.com/alesr/impact/cmd/impact@latest
```

## Requirements

- Go `1.25+`
- Terraform `1.6+` for plan workflows
- Scaleway credentials for API-backed commands

## Environment Variables

| Variable | Used by | Notes |
| --- | --- | --- |
| `SCW_ACCESS_KEY` | `impact actual`, `impact doctor`, Terraform provider | API access key |
| `SCW_SECRET_KEY` | `impact actual`, `impact doctor`, Terraform provider | API secret key/token |
| `SCW_ORGANIZATION_ID` | `impact actual`, `impact doctor` | Organization UUID |
| `IMPACT_SCW_API_BASE_URL` | API-backed commands | Optional base URL override (default `https://api.scaleway.com`) |

## Quick Start

### 1) Forecast from Terraform plan JSON

Before running, set real values for `db_password` and `redis_password` in `examples/terraform.tfvars.example`.

```bash
terraform -chdir=examples init
terraform -chdir=examples plan -var-file=terraform.tfvars.example -out=tfplan
terraform -chdir=examples show -json tfplan > examples/tfplan.json

impact plan --file examples/tfplan.json --format table
```

TUI mode:

```bash
impact plan --file examples/tfplan.json --tui
```

Directly from Terraform (run inside your Terraform directory):

```bash
impact plan --from-terraform --format table
```

### 2) Query measured impact

```bash
export SCW_SECRET_KEY="..."
export SCW_ORGANIZATION_ID="..."
export SCW_ACCESS_KEY="SCWXXXXXXXXXXXXXXXXX"

impact actual --start 2026-01-01 --end 2026-01-31 --format table
```

Optional filters:

- `--project`
- `--region`
- `--zone`
- `--service-category`
- `--product-category`

Accepted values:

- service categories: `baremetal`, `compute`, `storage`
- product categories: `applesilicon`, `blockstorage`, `dedibox`, `elasticmetal`, `instances`, `objectstorage`

Filter value normalization is supported, so separators like `-`, `_`, and spaces are accepted (for example `apple-silicon`, `apple_silicon`, `apple silicon`).

### 3) Run diagnostics

```bash
impact doctor
```

`doctor` checks:

- config/env visibility
- catalog endpoint reachability
- footprint query reachability (when auth and org are present)

## Plan Estimation Semantics

`impact plan` estimates monthly deltas with action-aware behavior:

- `create`: adds estimated monthly impact
- `delete`: subtracts estimated monthly impact
- `update`: estimates delta as before vs after (old config subtracted, new config added)
- `replace` (`delete` + `create`): modeled as delete + create transitions

Notes:

- `N/A` means footprint data is missing for a mapped product, not zero impact.
- totals can be partial when one or more rows have unknown footprint values.

## Showcase Configuration

Files:

- `examples/showcase.tf`
- `examples/terraform.tfvars.example`

Showcase includes:

- instance server replicas and block volume
- load balancer
- rdb and redis
- intentionally unsupported resources (`scaleway_instance_ip`, `scaleway_vpc_private_network`) to demonstrate diagnostics output

Example run from repo root:

```bash
terraform -chdir=examples init
terraform -chdir=examples plan -var-file=terraform.tfvars.example -out=tfplan
terraform -chdir=examples show -json tfplan > examples/tfplan.json

impact plan --file examples/tfplan.json --format table

# optional TUI view
impact plan --file examples/tfplan.json --tui
```

## Development

Run tests:

```bash
go test ./...
```

For local development without installing the binary:

```bash
go run ./cmd/impact --help
```

## Links

| Topic | Link |
| --- | --- |
| Environmental Footprint docs | https://www.scaleway.com/en/docs/environmental-footprint/ |
| Environmental Footprint integration guide | https://www.scaleway.com/en/docs/environmental-footprint/additional-content/environmental-footprint-integration/ |
| Environmental Footprint User API | https://www.scaleway.com/en/developers/api/environmental-footprint/user-api/ |
| Product Catalog Public Catalog API | https://www.scaleway.com/en/developers/api/product-catalog/public-catalog-api/ |
| Scaleway Go SDK | https://github.com/scaleway/scaleway-sdk-go |
