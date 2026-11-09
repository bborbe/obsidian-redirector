---
status: prompted
approved: "2026-11-09T00:46:24Z"
generating: "2026-11-09T01:33:19Z"
prompted: "2026-11-09T01:40:24Z"
branch: dark-factory/redirect-cache-control
---

## Summary

- The redirector converts an https link into an Obsidian deeplink and answers with a redirect.
- It decides per request whether the target is allowed, then sends the redirect with no instruction about storage.
- Anything between the caller and the service may therefore keep that answer and hand it to someone else.
- The redirect surface must tell every intermediary that its answers are single-use.

## Problem

The redirect target is derived from two request parameters and is emitted only after both pass an allowlist check. That check is a per-request decision. Nothing on the response tells a browser, proxy, or CDN that the decision must not be reused.

Two mechanisms make that a real risk rather than a theoretical one. First, a cache whose key ignores the query string answers a later, different request from the stored response. Second, a response cached **before** an allowlist entry was removed keeps serving a target the service no longer permits — the allowlist changes, the cache does not.

The failure is silent in the same way a wrong `+`-vs-`%20` encoding is silent: the link still opens, the headers look unremarkable, and only the caching layer's behaviour differs.

**Assumption**: the public hostname is fronted by a shared cache (browser, proxy, or CDN). Without one the defect is latent, not absent — the response is still not marked non-cacheable, and the first cache added in front of the service inherits it.

## Goal

Every response the `/obsidian` handler produces is explicitly non-cacheable, so no intermediary can replay a redirect or a rejection that this service did not decide for that request.

## Non-goals

- Changing the allowlist logic, the accepted parameters, or the emitted deeplink format.
- Adding a cache directive to the admin endpoints (`/healthz`, `/readiness`, `/metrics`, `/setloglevel`, `/gc`, `/testloglevel`, `/sentryalert`) — this work is scoped to `/obsidian`.
- Adding or removing any other response header.
- Introducing any caching layer, positive or negative.
- Purging any cache that already holds a pre-change response — that is an operator step, recorded as a failure mode, not code.
- Changing the status codes (`302` for a redirect, `400` for a rejection).

## Acceptance Criteria

- [ ] A valid `GET /obsidian?vault=<allowed>&file=<allowed>` returns `302`, a `Location` equal to the encoded deeplink, and `Cache-Control: no-store`. Evidence shape: HTTP response status + headers, asserted by a Ginkgo example in `pkg/handler/obsidian-redirect_test.go` reading `resp.Result().Header`.
- [ ] Every rejection branch of `/obsidian` — unknown vault, disallowed file prefix, empty parameter, absolute or traversing file, a `url` parameter, and control characters in either parameter — returns `400` and `Cache-Control: no-store`. Evidence shape: HTTP response status + headers, asserted per branch by Ginkgo examples reading `resp.Result().Header`.
- [ ] The header assertions are a regression lock: removing the `Set` call makes the handler suite fail. Evidence shape: revert the implementation line, run `go test ./pkg/handler/...`, observe a non-zero exit; restore it, observe a pass, and confirm `git status --porcelain` is empty.
- [ ] No `Location` expectation was weakened to make the suite pass. Evidence shape: negative evidence — `git diff -U0 pkg/handler/obsidian-redirect_test.go` shows no removed line containing `obsidian://open`.
- [ ] `make precommit` passes. Evidence shape: command exit status.

## Verification

```bash
go test ./pkg/handler/...
make precommit
```

## Desired Behavior

1. A successful redirect from `/obsidian` carries `Cache-Control: no-store`.
2. A rejected request to `/obsidian` carries `Cache-Control: no-store`.
3. The directive is present on the response whether the status is `302` or `400`, and it is set before the status is written.
4. The handler suite asserts the header on both the redirect and every rejection branch, so the behaviour cannot be removed silently.

## Constraints

- The emitted `Location` stays byte-for-byte the encoded deeplink; `http.Redirect` is still not used.
- No new dependency.
- The existing rejection body text is unchanged.
- Existing tests keep passing without modification beyond adding the new assertions.

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---|---|---|
| Directive set on the redirect branch but not on the rejection path | The rejection AC fails | Set it in `writeRejection`, which both rejection branches already share |
| Header written after the status is committed | The directive is absent on the wire while an assertion reading the live header map still passes | Assert on `resp.Result().Header`, which snapshots at commit time; move the `Set` above the status write |
| A shared cache already stored a `/obsidian` response before this change | `no-store` does not evict it; the stored entry replays until its original TTL expires | Purge the CDN/proxy cache for `/obsidian*`, then confirm `curl -sI` shows `cache-control: no-store` |
| A new rejection branch added later that bypasses `writeRejection` | A cacheable rejection ships | Route the branch through `writeRejection` |

## Security / Abuse Cases

The redirect target is built only from allowlisted values, and the service never emits a caller-supplied URL. Marking the response non-cacheable keeps that per-request guarantee intact at any shared cache between Telegram and this service; without it, a cache is a second, unvalidated source of redirect targets — one that survives an allowlist change.

## Do-Nothing Option

Leave the responses cacheable. The service keeps working and the redirect keeps opening, which is why this survives review: the defect only appears when a cache sits in front of the service, and then it appears as a redirect that was never validated for the request that received it.
