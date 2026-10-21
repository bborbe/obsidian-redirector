# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

## Unreleased

- chore: update github.com/bborbe/log to v1.7.5, github.com/bborbe/metrics to v0.6.8, github.com/bborbe/service to v1.11.1, github.com/bborbe/time to v1.27.18

## v0.1.7

- chore: update Go to 1.27.3 and github.com/bborbe/sentry to v1.10.5, github.com/bborbe/service to v1.11.0, github.com/bborbe/time to v1.27.17, github.com/bborbe/validation to v1.5.5

## v0.1.6

- chore: update go module dependencies

## v0.1.5

- chore: update Go to 1.27.2 and github.com/bborbe/errors to v1.6.3, github.com/bborbe/http to v1.26.30, github.com/bborbe/log to v1.7.4, github.com/bborbe/metrics to v0.6.7, github.com/bborbe/run to v1.11.2, github.com/bborbe/sentry to v1.10.4, github.com/bborbe/service to v1.10.16, github.com/bborbe/time to v1.27.16, github.com/bborbe/validation to v1.5.4, github.com/getsentry/sentry-go to v0.50.0, github.com/onsi/ginkgo/v2 to v2.33.1, github.com/prometheus/client_golang to v1.25.0

## v0.1.4

- fix: bump `osv-scanner` to v2.6.0 and `golang.org/x/net` to v0.60.0 so the Linux vulnerability gates stop failing. v2.3.1 pins `golang.org/x/tools` v0.38.0, whose SSA builder aborts with `unexpected expr: *ast.KeyValueExpr` on the promoted-field composite-literal key Go 1.27 permits in the Linux stdlib, so a repo on the old pin passes locally on darwin and fails only in Linux CI. `x/net` v0.58.0 carries `GO-2026-6603/6610/6611/6612/6617`, which fail both `vulncheck` and `trivy`.

## v0.1.3

- chore: update github.com/onsi/gomega to v1.44.0

## v0.1.2

- fix: normalise the file-prefix allowlist so an entry written without a trailing `/` cannot widen the match. `FileAllowlist.Contains` is a prefix match, so `25 Tasks` also admitted `25 TasksExtra/x.md`; the boundary was previously held only by the convention that every configured entry happens to end in a separator. Normalisation lives in a new `ParseFileAllowlist` seam rather than in `ParseAllowlist`, which is shared with the vault list — appending `/` there would stamp it onto vault names and, since `VaultAllowlist.Contains` is exact membership, reject every request

## v0.1.1

- fix: Correct the copyright year in `pkg/factory/factory.go` from 2025 to 2026, matching the other files this PR touches
- docs: Add the `LICENSE` file that every source header already references

## v0.1.0

- feat: add `GET /obsidian?vault=<v>&file=<p>`, answering 302 with the equivalent `obsidian://open` deeplink so a URL Telegram will carry resolves into the vault on the tap. Telegram rejects a `text_link` whose URL uses a custom scheme, so the deeplink cannot be attached to a message directly.
- feat: percent-encode the emitted `file` with `%20` for spaces rather than `+` — Obsidian does not decode `+` in a query string, so a link built that way opens nothing while looking correct
- feat: constrain what the endpoint will emit with a vault allowlist and a vault-relative path-prefix allowlist; an empty allowlist allows nothing, so a missing configuration fails closed
- feat: reject control characters in `vault` and `file`, closing a CRLF-injection path into the `Location` header
- feat: serve the endpoint on a public listener (`:8080`) separate from the canonical admin server (`:9090`), so an unauthenticated redirect is never mounted beside the admin handlers
- fix: `url` parameter is rejected outright rather than ignored, so the open-redirect guard fails loudly if a passthrough is ever reintroduced

## v0.0.1

- Initial commit
