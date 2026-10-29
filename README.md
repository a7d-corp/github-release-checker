# github-release-checker

A terminal UI that shows the latest GitHub release of each repository in a list. For each repository, it also shows the number of commits on the default branch since that release.

## Install

```sh
go install github.com/a7d-corp/github-release-checker@latest
```

## Configuration

The tool reads the list of repositories from `$HOME/.config/github-release-checker/repositories.yaml`:

```yaml
---
repos:
  - giantswarm/cloud-provider-cloud-director-app
  - giantswarm/cloud-provider-proxmox-app
  - giantswarm/cloud-provider-vsphere-app
```

To use a different file, use the `--config` flag:

```sh
github-release-checker --config ./repositories.yaml
```

## Authentication

Set `GITHUB_TOKEN` or `GH_TOKEN` to a GitHub token. The tool uses `GITHUB_TOKEN` first.

## Usage

The table has these columns:

| Column        | Value                                                     |
|---------------|-----------------------------------------------------------|
| Repository    | The repository in `owner/name` form.                      |
| Release tag   | The tag of the latest release. Drafts and pre-releases are not included. |
| Release date  | The publish date of the latest release.                   |
| Commits since | The number of commits on the default branch since the release. |

The repository with the most commits since its release is at the top. Rows without a commit count go last.

A repository with no release shows `none`. An API failure shows `error` in the row and the message under the table.

Keys:

- `↑`/`↓`: scroll the table.
- `r`: check all repositories again.
- `q` or `ctrl+c`: quit.
