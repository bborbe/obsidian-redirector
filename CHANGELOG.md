# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

## Unreleased

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
