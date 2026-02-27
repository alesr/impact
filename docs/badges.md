# Badges

This project supports an (also experimental) automated badge that publishes estimated monthly impact from Terraform plans.

Current example badge:

![Impact estimate](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/alesr/impact/main/badges/impact-estimate.json)

## How it works

The workflow in `.github/workflows/impact-estimate-badge.yml` runs this pipeline:

1. Generate Terraform plan data in CI:
   - `terraform -chdir=examples init`
   - `terraform -chdir=examples plan ... -out=tfplan`
   - `terraform -chdir=examples show -json tfplan > examples/tfplan.json`
2. Inspect the generated plan with IMPACT:
   - `impact plan --file examples/tfplan.json --format json`
3. Convert the impact JSON into a Shields endpoint payload:
   - `go run ./cmd/impact-badge --input .tmp/impact.json --output badges/impact-estimate.json`
4. Commit updated `badges/impact-estimate.json` when content changes.

In short: Terraform plan inspection -> IMPACT estimate JSON -> badge endpoint JSON.

## CI requirements

Set these repository secrets in GitHub Actions:

- `SCW_ACCESS_KEY`
- `SCW_SECRET_KEY`
- `SCW_ORGANIZATION_ID`

## Local generation

You can generate the same badge payload locally:

```bash
mkdir -p .tmp
impact plan --file examples/tfplan.json --format json > .tmp/impact.json
go run ./cmd/impact-badge --input .tmp/impact.json --output badges/impact-estimate.json
```
