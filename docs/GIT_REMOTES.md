# Git repository destinations

This project can be tracked in multiple Git remotes. Typical layout:

| # | Host | Role |
|---|------|------|
| 1 | **GitHub** (`eduwallet/oauth2-server`) | Primary public upstream, CI/CD badges, issues (see README). |
| 2 | **GitHub** (`HarryKodden/oauth2-server`) | Alternate / personal fork; Helm Argo CD example uses this URL by default. |
| 3 | **Forgejo** ([`git.homelab.kodden.nl/harry/oauth2-server`](https://git.homelab.kodden.nl/harry/oauth2-server)) | Private homelab mirror; same history as GitHub when mirroring is enabled. |

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

On each push to `main`, `develop`, or tags `v*`, the workflow pushes the same ref to Forgejo **if** the repository secret is set:

| Name | Required | Description |
|------|----------|-------------|
| `FORGEJO_TOKEN` | Yes, to enable mirror | Forgejo personal access token with **write** access to the target repo. |

Optional GitHub **repository variable**:

| Name | Default |
|------|---------|
| `FORGEJO_MIRROR_URL` | `https://git.homelab.kodden.nl/harry/oauth2-server.git` |

If the secret is absent, the job exits successfully without pushing (forks stay quiet until you add the secret).

## Argo CD from homelab

To deploy from Forgejo instead of GitHub, point `repoURL` at the homelab clone URL. See [`helm/oauth2-server/argocd/application-homelab-forgejo.yaml`](../helm/oauth2-server/argocd/application-homelab-forgejo.yaml).

Container images are published to **GHCR** only (`ghcr.io/<owner>/<repo>`). Pull from there in homelab deployments, for example:

`docker pull ghcr.io/harrykodden/oauth2-server:latest`
