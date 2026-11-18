# Git repository destinations

This project is tracked in multiple Git remotes. Canonical layout:

| # | Host | Role |
|---|------|------|
| 1 | **GitHub** (`eduwallet/oauth2-server`) | **Canonical** public upstream (`origin`). Releases, CI badges, and GHCR images. |
| 2 | **GitHub** (`HarryKodden/oauth2-server`) | Personal mirror (`harrykodden` remote). Keep tags aligned with eduwallet. |
| 3 | **Forgejo** ([`git.homelab.kodden.nl/harry/oauth2-server`](https://git.homelab.kodden.nl/harry/oauth2-server)) | Private homelab mirror; pushed by Actions when `FORGEJO_TOKEN` is set. |

## Local remotes

```bash
git remote -v
# origin       git@github.com:eduwallet/oauth2-server.git
# harrykodden  git@github.com:HarryKodden/oauth2-server.git
```

Set this up on a fresh clone:

```bash
git clone git@github.com:eduwallet/oauth2-server.git
cd oauth2-server
git remote add harrykodden git@github.com:HarryKodden/oauth2-server.git
```

Version tags must mean the **same commit** on every remote. Never reuse a tag name that already exists on eduwallet for a different commit. Use `make tag` / `scripts/tag-version.sh` (pushes `origin`, then `harrykodden` if present).

## Manual: add the homelab remote

Create an empty repository under your Forgejo user/org, then:

```bash
git remote add homelab https://git.homelab.kodden.nl/harry/oauth2-server.git
git fetch origin
git push homelab main
git push homelab develop   # if you use it
git push homelab --tags
```

Use a [personal access token](https://git.homelab.kodden.nl/user/settings/applications) as the password when prompted over HTTPS, or configure SSH.

## Automated mirror from GitHub Actions

Workflow: [`.github/workflows/mirror-homelab-forgejo.yml`](../.github/workflows/mirror-homelab-forgejo.yml)

On each push to `main`, `develop`, or tags `v*` **on the repo where the workflow runs** (preferably eduwallet), the job pushes the same ref to Forgejo **if** the repository secret is set:

| Name | Required | Description |
|------|----------|-------------|
| `FORGEJO_TOKEN` | Yes, to enable mirror | Forgejo personal access token with **write** access to the target repo. |

Optional GitHub **repository variable**:

| Name | Default |
|------|---------|
| `FORGEJO_MIRROR_URL` | `https://git.homelab.kodden.nl/harry/oauth2-server.git` |

If the secret is absent, the job exits successfully without pushing.

Ensure `FORGEJO_TOKEN` is configured on **eduwallet/oauth2-server** so homelab tracks the canonical releases.

## Argo CD from homelab

To deploy from Forgejo instead of GitHub, point `repoURL` at the homelab clone URL. See [`helm/oauth2-server/argocd/application-homelab-forgejo.yaml`](../helm/oauth2-server/argocd/application-homelab-forgejo.yaml).

Container images are published to **GHCR** from the repo that runs `release.yml`. Prefer the eduwallet image:

`docker pull ghcr.io/eduwallet/oauth2-server:latest`
