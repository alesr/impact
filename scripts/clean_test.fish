#!/usr/bin/env fish

function run_step --argument-names message command
    echo "==> $message"
    fish -c "$command"
    if test $status -ne 0
        echo "Failed: $message"
        exit 1
    end
end

run_step "Cleaning generated Terraform artifacts" "rm -f examples/tfplan examples/tfplan.json"

run_step "Terraform init" "terraform -chdir=examples init"
run_step "Terraform plan" "terraform -chdir=examples plan -var-file=terraform.tfvars.example -out=tfplan"
run_step "Terraform show JSON" "terraform -chdir=examples show -json tfplan > examples/tfplan.json"

run_step "go list" "go list ./..."
run_step "go build" "go build ./..."
run_step "go test" "go test -race -count=1 ./..."
run_step "go vet" "go vet ./..."
run_step "go mod verify" "go mod verify"

run_step "impact doctor" "go run ./cmd/impact doctor"
run_step "impact plan (table)" "go run ./cmd/impact plan --file examples/tfplan.json --format table"
run_step "impact actual (json)" "go run ./cmd/impact actual --start 2026-01-01 --end 2026-01-31 --format json"

echo "==> Done"
