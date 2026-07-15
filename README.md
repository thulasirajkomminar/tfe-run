# tfe-run

A Go CLI to trigger Terraform runs on multiple Terraform Enterprise (TFE) / HCP Terraform workspaces using the official [go-tfe](https://github.com/hashicorp/go-tfe) API client.

## Prerequisites

### Go

```bash
brew install go
```

### Terraform Credentials

Configure your credentials using one of the following methods (checked in order):

1. **Environment variable** — set `TFE_TOKEN`
2. **~/.terraformrc** — credentials block with token
3. **~/.terraform.d/credentials.tfrc.json** — JSON credentials file

See: [https://developer.hashicorp.com/terraform/cli/config/config-file#credentials-1](https://developer.hashicorp.com/terraform/cli/config/config-file#credentials-1)

## Install

```bash
go install github.com/thulasirajkomminar/tfe-run@latest
```

Or build from source:

```bash
git clone https://github.com/thulasirajkomminar/tfe-run.git
cd tfe-run
go install
```

## Environment Variables

| Variable    | Description                                      |
|-------------|--------------------------------------------------|
| `TFE_TOKEN` | API token for TFE/HCP Terraform                  |
| `TFE_ORG`   | Default organization (overridden by `--org` flag)|

## Usage

### Trigger runs by workspace tags

```bash
tfe-run --org myorg --tags "production,app1" --planonly true
```

By default, only workspaces carrying **every** given tag are selected. Use `--tagmatch any` to select workspaces matching **any** of the tags:

```bash
tfe-run --org myorg --tags "production,app1" --tagmatch any
```

### Preview matching workspaces (dry run)

Use `--dry-run` to see which workspaces would be selected — with their tags — without triggering any runs:

```bash
tfe-run --org myorg --tags "production,app1" --dry-run
```

### Trigger runs by workspace names

```bash
tfe-run --org myorg --workspace "ws-app1,ws-app2" --planonly false
```

### Plan-only behavior

| `--planonly` value | Behavior                             |
|--------------------|--------------------------------------|
| `true`             | Speculative plan-only run            |
| `false`            | Full apply run                       |
| _(empty/omitted)_  | Uses the workspace's default setting |

### Using environment variables instead of flags

```bash
export TFE_TOKEN="your-token-here"
export TFE_ORG="myorg"

tfe-run --tags "staging,network"
```

## Flags

```bash
--dry-run            List the workspaces that would be run, without triggering anything
--org string         TFE/HCP Terraform organization name (or set TFE_ORG env var)
--tags string        Comma-separated workspace tags to filter by
--tagmatch string    Tag matching mode: "all" (workspace has every tag) or "any" (workspace has at least one tag) (default "all")
--workspace string   Comma-separated workspace names
--planonly string    Plan only run: true/false (empty = workspace default)
-h, --help           Help for tfe-run
```

> [!Note]
> `--tags` and `--workspace` are mutually exclusive — use one or the other.
