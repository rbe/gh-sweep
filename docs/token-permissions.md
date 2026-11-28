# Token permissions of the gh-sweep endpoints

This table lists each REST endpoint that gh-sweep calls through `gh api`, the fine-grained permission it needs, and the scope text for a classic token. The README section "Token permissions" gives the same facts for each flag.

Sources, read on 2026-11-28: github/docs `src/github-apps/data/fpt-2022-11-28/fine-grained-pat.json` and `fine-grained-pat-permissions.json` (fine-grained token), github/rest-api-description `descriptions/api.github.com/api.github.com.json` (classic scope text).

| Endpoint | Fine-grained token | Fine-grained permission | Classic scope text |
| --- | --- | --- | --- |
| GET `/repos/{owner}/{repo}/actions/artifacts` | yes | Repository: Actions read | OAuth app tokens and personal access tokens (classic) need the `repo` scope to use this endpoint with a private repository. |
| DELETE `/repos/{owner}/{repo}/actions/artifacts/{artifact_id}` | yes | Repository: Actions write | OAuth tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}/actions/caches` | yes | Repository: Actions read | OAuth tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| DELETE `/repos/{owner}/{repo}/actions/caches/{cache_id}` | yes | Repository: Actions write | OAuth tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| DELETE `/repos/{owner}/{repo}/actions/caches` | yes | Repository: Actions write | OAuth tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}/actions/cache/usage` | yes | Repository: Actions read | OAuth tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}/actions/cache/retention-limit` | yes | Repository: Administration read | OAuth tokens and personal access tokens (classic) need the `admin:repository` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}/actions/cache/storage-limit` | yes | Repository: Actions read | OAuth tokens and personal access tokens (classic) need the `admin:repository` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}/actions/permissions/artifact-and-log-retention` | yes | Repository: Administration read | OAuth app tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| PUT `/repos/{owner}/{repo}/actions/permissions/artifact-and-log-retention` | yes | Repository: Administration write | OAuth app tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}/actions/permissions/workflow` | yes | Repository: Administration read | OAuth tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| PUT `/repos/{owner}/{repo}/actions/permissions/workflow` | yes | Repository: Administration write | OAuth app tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}/actions/runs` | yes | Repository: Actions read | OAuth app tokens and personal access tokens (classic) need the `repo` scope to use this endpoint with a private repository. |
| DELETE `/repos/{owner}/{repo}/actions/runs/{run_id}` | yes | Repository: Actions write | OAuth tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| DELETE `/repos/{owner}/{repo}/actions/runs/{run_id}/logs` | yes | Repository: Actions write | OAuth tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| GET `/user/packages/{package_type}/{package_name}/versions` | no | - | OAuth app tokens and personal access tokens (classic) need the `read:packages` scope to use this endpoint. |
| DELETE `/user/packages/{package_type}/{package_name}/versions/{package_version_id}` | no | - | OAuth app tokens and personal access tokens (classic) need the `read:packages` and `delete:packages` scopes to use this endpoint. |
| GET `/orgs/{org}/packages/{package_type}/{package_name}/versions` | no | - | OAuth app tokens and personal access tokens (classic) need the `read:packages` scope to use this endpoint. |
| DELETE `/orgs/{org}/packages/{package_type}/{package_name}/versions/{package_version_id}` | no | - | OAuth app tokens and personal access tokens (classic) need the `read:packages` and `delete:packages` scopes to use this endpoint. |
| GET `/repos/{owner}/{repo}/releases` | yes | Repository: Contents read | - |
| DELETE `/repos/{owner}/{repo}/releases/{release_id}` | yes | Repository: Contents write | - |
| DELETE `/repos/{owner}/{repo}/releases/assets/{asset_id}` | yes | Repository: Contents write | - |
| GET `/user/codespaces` | yes | Repository: Codespaces read | OAuth app tokens and personal access tokens (classic) need the `codespace` scope to use this endpoint. |
| DELETE `/user/codespaces/{codespace_name}` | yes | Repository: Codespaces write | OAuth app tokens and personal access tokens (classic) need the `codespace` scope to use this endpoint. |
| GET `/users/{username}/settings/billing/usage/summary` | yes | User: Plan read | - |
| GET `/repos/{owner}/{repo}/pulls` | yes | Repository: Pull requests read | - |
| GET `/repos/{owner}/{repo}/branches` | yes | Repository: Contents read | - |
| DELETE `/repos/{owner}/{repo}/git/refs/{ref}` | yes | Repository: Contents write | - |
| GET `/repos/{owner}/{repo}/deployments` | yes | Repository: Deployments read | - |
| POST `/repos/{owner}/{repo}/deployments/{deployment_id}/statuses` | yes | Repository: Deployments write | OAuth app tokens and personal access tokens (classic) need the `repo_deployment` scope to use this endpoint. |
| DELETE `/repos/{owner}/{repo}/deployments/{deployment_id}` | yes | Repository: Deployments write | OAuth app tokens and personal access tokens (classic) need the `repo` or `repo_deployment` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}/environments` | yes | Repository: Actions read | OAuth app tokens and personal access tokens (classic) need the `repo` scope to use this endpoint with a private repository. |
| DELETE `/repos/{owner}/{repo}/environments/{environment_name}` | yes | Repository: Administration write | OAuth app tokens and personal access tokens (classic) need the `repo` scope to use this endpoint. |
| GET `/repos/{owner}/{repo}` | yes | Repository: Metadata read | - |
| PATCH `/repos/{owner}/{repo}` | yes | Repository: Administration write | - |

`gh cache list`, `gh cache delete`, `gh repo list`, and `gh repo view` call these endpoints or the GraphQL API of GitHub. No test of this project calls GitHub, so no test proves the table. A Packages endpoint with "no" in the second column does not accept a fine-grained token.
