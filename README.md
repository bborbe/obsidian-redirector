# obsidian-redirector

Single-purpose HTTPS redirector: turns an `https://` URL Telegram will carry into an `obsidian://` deeplink, so a delivered escalation is one tap from the note it is about.

Telegram rejects a `text_link` whose URL uses a custom scheme (`Unsupported URL protocol`), so the deeplink cannot be attached directly. This service accepts the link as `https://redirect.benjamin-borbe.de/obsidian?vault=<v>&file=<p>` and returns a `302` whose `Location` is the equivalent `obsidian://open` URL.

## Run locally

```bash
make test
make run
```

## Endpoint

```
GET /obsidian?vault=<v>&file=<p>  →  302  Location: obsidian://open?vault=<v>&file=<p>
```

`vault` and `file` are validated against allowlists. The endpoint **only ever** emits an `obsidian://` target — there is deliberately no `?url=` passthrough, which would be an open redirect.

## Deploy

Deployed as a raw-manifest component of the [nuke](https://github.com/bborbe/nuke) repo, alongside `notification-telegram`:

```bash
make buca
```
