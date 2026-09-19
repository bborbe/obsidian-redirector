# Changelog

## Unreleased

- feat: add `GET /obsidian?vault=<v>&file=<p>`, answering 302 with the equivalent `obsidian://open` deeplink so a URL Telegram will carry resolves into the vault on the tap. Telegram rejects a `text_link` whose URL uses a custom scheme, so the deeplink cannot be attached to a message directly.
- feat: percent-encode the emitted `file` with `%20` for spaces rather than `+` — Obsidian does not decode `+` in a query string, so a link built that way opens nothing while looking correct
- feat: constrain what the endpoint will emit with a vault allowlist and a vault-relative path-prefix allowlist; an empty allowlist allows nothing, so a missing configuration fails closed
- feat: reject control characters in `vault` and `file`, closing a CRLF-injection path into the `Location` header
- feat: serve the endpoint on a public listener (`:8080`) separate from the canonical admin server (`:9090`), so an unauthenticated redirect is never mounted beside the admin handlers
- fix: `url` parameter is rejected outright rather than ignored, so the open-redirect guard fails loudly if a passthrough is ever reintroduced

## v0.0.1

- Initial commit
