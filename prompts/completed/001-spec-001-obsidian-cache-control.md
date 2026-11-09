---
status: completed
spec: [001-redirect-cache-control]
summary: 'Added Cache-Control: no-store to every /obsidian response before status commit, with committed-header Ginkgo assertions and a changelog entry.'
execution_id: 9617fa69-5fad-51e0-8b96-ea11dde8c10b-exec-001-spec-001-obsidian-cache-control
dark-factory-version: dev
created: "2026-11-09T01:34:30Z"
queued: "2026-11-09T01:47:32Z"
started: "2026-11-09T01:47:34Z"
completed: "2026-11-09T01:57:29Z"
branch: dark-factory/redirect-cache-control
---

<summary>
- Successful Obsidian deeplink redirects now tell browsers, proxies, and CDNs not to store the answer.
- Rejected requests get the same non-cacheable marking.
- The marking is applied before the response status is committed, so it actually reaches the wire.
- Existing redirect behavior, status codes, and link contents are unchanged.
- Every rejection path is covered: unknown vault, disallowed path prefix, empty parameter, absolute path, traversing path, a `url` parameter, and control characters in either parameter.
- Tests assert the marking on the real committed response headers rather than the in-memory header map.
- Removing the marking would make the handler test suite fail, so it cannot be dropped silently.
- No other endpoints or response headers are touched.
- A changelog entry records the fix.
</summary>

<objective>
Make every response from the `/obsidian` handler explicitly non-cacheable by emitting `Cache-Control: no-store` before the status is written, on both the `302` redirect branch and every `400` rejection branch. Lock the behavior with Ginkgo assertions that read the committed response headers, so no intermediary can replay a redirect or rejection this service did not decide for that request.
</objective>

<context>
This repo has no `CLAUDE.md`. Project conventions live in `docs/dod.md`; read it before changing code.

Read these files before making changes (paths are relative to the repo root):
- `pkg/handler/obsidian-redirect.go` — the handler under change: `NewObsidianRedirectHandler`, `writeRejection`, and the `Location` write in the redirect branch.
- `pkg/handler/obsidian-redirect_test.go` — the Ginkgo suite to extend; note the existing `request(rawQuery string) *httptest.ResponseRecorder` helper and the existing `Location` expectations.
- `pkg/handler/handler_suite_test.go` — suite bootstrap (`TestSuite`, Ginkgo/Gomega wiring).
- `docs/dod.md` — definition of done (Ginkgo/Gomega, `make precommit`).
- `CHANGELOG.md` — the frozen preamble and `## Unreleased` section.

Background from the coding plugin (in-container paths):
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md` — Ginkgo/Gomega conventions.
- `/home/node/.claude/plugins/marketplaces/coding/docs/changelog-guide.md` — conventional-prefix changelog rules and the frozen preamble.

Why `resp.Result().Header` matters: `httptest.ResponseRecorder` clones its header map into `snapHeader` at the first `WriteHeader`/`Write`. Reading `resp.Result().Header` therefore inspects the headers as committed, and catches a `Set` placed after the status write — which the live `resp.Header()` map would still show as present.
</context>

<requirements>
1. In `pkg/handler/obsidian-redirect.go`, add an unexported constant next to the existing `obsidianScheme` constant (do not export it, do not add a new dependency):

```go
// cacheControlNoStore is the Cache-Control directive applied to every
// /obsidian response. The redirect target is a per-request allowlist decision,
// so no intermediary may replay the answer for a different request.
const cacheControlNoStore = "no-store"
```

2. In `writeRejection`, set the header as the FIRST statement, before `WriteHeader`:

```go
func writeRejection(resp http.ResponseWriter, message string) {
	resp.Header().Set("Cache-Control", cacheControlNoStore)
	resp.WriteHeader(http.StatusBadRequest)
	if _, err := resp.Write([]byte(message)); err != nil {
		glog.V(2).Infof("write rejection response failed: %v", err)
	}
}
```

Both rejection branches (`url` parameter and `validateRedirectRequest` failure) already call `writeRejection`, so this one edit covers both. Do not duplicate the `Set` at the call sites.

3. In the success branch of `NewObsidianRedirectHandler`, set the header before the status is written. The existing lines are:

```go
resp.Header().Set("Location", target)
resp.WriteHeader(http.StatusFound)
```

Change them to:

```go
resp.Header().Set("Cache-Control", cacheControlNoStore)
resp.Header().Set("Location", target)
resp.WriteHeader(http.StatusFound)
```

Do not replace the direct `WriteHeader`/`Location` write with `http.Redirect`; the `Location` must stay byte-for-byte the encoded deeplink.

4. Extend `pkg/handler/obsidian-redirect_test.go`. Every new assertion MUST read the committed response headers:

```go
Expect(resp.Result().Header.Get("Cache-Control")).To(Equal("no-store"))
```

Add this assertion to the following existing `It` blocks (all already exist in the file):
- Under `Context("a valid request")`, add a new `It("marks the redirect as non-cacheable", ...)` that performs `request("vault=Personal&file=25%20Tasks%2FMy%20Note")` and asserts `Cache-Control` is `no-store`. Also keep asserting (or rely on the existing sibling examples for) the `302` status.
- `rejects a vault outside the allowlist with 400` — unknown vault branch.
- `rejects a file outside the path allowlist with 400` — disallowed file prefix branch.
- `rejects a file that traverses upward with 400` — traversing branch.
- `rejects an absolute file with 400` — absolute branch.
- `rejects a file containing CRLF, which would otherwise reach the Location header` — control characters in `file`.
- `rejects a vault containing CRLF` — control characters in `vault`.
- `rejects a missing vault with 400` — empty-parameter branch.
- `rejects a missing file with 400` — empty-parameter branch.
- `rejects a url parameter outright with 400` — `url` parameter branch.

Also extend the open-redirect falsifier example `returns a rejection status for every hostile input` so that each query in `hostileQueries` additionally asserts `resp.Result().Header.Get("Cache-Control")` equals `no-store`. This covers the `vault=&file=` and empty-query cases explicitly.

Use the existing `request` helper; do not add a second HTTP round-trip or a new helper unless strictly necessary.

5. The new assertions are the regression lock. Removing either `resp.Header().Set("Cache-Control", cacheControlNoStore)` call MUST make `go test ./pkg/handler/...` fail. Do not add skips, tolerances, or conditional guards that let the suite pass without the header.

6. Do NOT weaken or delete any existing `Location` assertion. Every line currently containing `obsidian://open` in the test file must remain. The only permitted test edits are additions of the `Cache-Control` assertions described above (and the one new valid-request example).

7. Add a changelog entry under the existing `## Unreleased` heading in `CHANGELOG.md`, using a conventional prefix and a flat bullet (do not touch the frozen preamble, do not add category subheadings):

```markdown
- fix: Mark /obsidian responses Cache-Control: no-store
```

8. Failure-mode handling to preserve:
- A rejection must never be cacheable: the `Set` lives in `writeRejection`, so any branch that routes through it is covered. Do not add a rejection path that writes `400` directly.
- The header must be set before the status write. `resp.Result().Header` snapshots at commit time; if a `Set` is placed after `WriteHeader`, the assertion must fail.
- If `resp.Write` of the rejection body fails, the existing log-and-continue behavior stays; the header is already committed by then.
</requirements>

<constraints>
- Do NOT change the allowlist logic, the accepted parameters, or the emitted deeplink format.
- Do NOT add a cache directive to the admin endpoints (`/healthz`, `/readiness`, `/metrics`, `/setloglevel`, `/gc`, `/testloglevel`, `/sentryalert`); this work is scoped to `/obsidian`.
- Do NOT add or remove any other response header. `Cache-Control: no-store` is the only header change.
- Do NOT introduce any caching layer, positive or negative.
- Do NOT purge any cache that already holds a pre-change response — that is an operator step, not code.
- Do NOT change the status codes (`302` for a redirect, `400` for a rejection).
- The emitted `Location` stays byte-for-byte the encoded deeplink; `http.Redirect` is still not used.
- No new dependency.
- The existing rejection body text (`"url parameter is not supported"`, `"invalid request"`) is unchanged.
- Existing tests keep passing without modification beyond adding the new assertions.
- Do NOT commit — dark-factory handles git.
</constraints>

<verification>
Run `go test ./pkg/handler/...` — must pass.
Run `make precommit` — must pass (format + lint + test + security checks).

Regression-lock check: temporarily delete the `resp.Header().Set("Cache-Control", cacheControlNoStore)` line from `writeRejection`, run `go test ./pkg/handler/...`, and confirm it fails; restore the line and confirm it passes. Repeat for the `Set` in the redirect branch. Leave both lines restored before finishing.

Confirm no `Location` expectation was weakened: every line containing `obsidian://open` that existed before this change is still present in `pkg/handler/obsidian-redirect_test.go`.
</verification>

<!--
OPEN QUESTIONS FOR THE HUMAN REVIEWER (not instructions for the execution agent):

1. Changelog scope. The spec does not mention a changelog entry. Requirement 7
   follows this repo's existing convention (every notable change has an
   `## Unreleased` bullet and `.maintainer.yaml` enables `autoRelease`). If the
   changelog is out of scope for this spec, drop requirement 7 at audit time.

2. Directive breadth. The spec fixes the directive to `no-store` and lists
   "adding or removing any other response header" as a non-goal. Some
   intermediaries also honor `no-cache` / `private`; this prompt deliberately
   does not add them because the spec forbids extra headers. No action unless
   the reviewer wants to widen the spec.

3. "Empty parameter" branch. The spec names "empty parameter" as a rejection
   branch; the existing suite covers it as separate "missing vault" and
   "missing file" examples plus the `vault=&file=` entry in `hostileQueries`.
   The prompt asserts the header on all three. No new branch is invented.
-->
