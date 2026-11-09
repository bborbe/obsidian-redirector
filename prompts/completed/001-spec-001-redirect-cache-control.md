---
status: completed
spec: [001-redirect-cache-control]
summary: 'Added Cache-Control: no-store to every /obsidian response and locked it in with Result()-snapshot assertions across redirect and all rejection branches.'
execution_id: 8c85c225-6bc1-534d-aa8a-2da55d0cbb10-exec-001-spec-001-redirect-cache-control
dark-factory-version: dev
created: "2026-11-09T02:00:06Z"
queued: "2026-11-09T02:07:04Z"
started: "2026-11-09T02:07:06Z"
completed: "2026-11-09T02:15:28Z"
branch: dark-factory/redirect-cache-control
---

<summary>
- Every `/obsidian` response now tells browsers, proxies, and CDNs that the answer is single-use.
- A successful redirect still returns `302` with the exact same encoded deeplink in `Location`.
- A rejected request still returns `400` with the same rejection body text.
- Both the redirect and every rejection branch carry `Cache-Control: no-store`.
- The directive is written before the status code, so it is present on the wire rather than only in the live header map.
- The handler test suite now locks this in: removing the header call fails the suite.
- Admin endpoints (`/healthz`, `/readiness`, `/metrics`, `/setloglevel`, `/gc`, `/testloglevel`, `/sentryalert`) are untouched.
- No new dependency, no status-code change, no allowlist change, no deeplink format change.
- Existing `Location` expectations are preserved byte-for-byte; none are weakened to make the suite pass.
</summary>

<objective>
Make every response produced by the `/obsidian` handler explicitly non-cacheable by setting `Cache-Control: no-store` before the status code is written, on both the redirect branch and the shared rejection path, and add Ginkgo assertions that fail if the header is removed.
</objective>

<context>
Read these before changing anything:
- `pkg/handler/obsidian-redirect.go` — the handler under change. Note `NewObsidianRedirectHandler` (success branch writes `Location` then `resp.WriteHeader(http.StatusFound)`), `writeRejection` (shared by the `url`-parameter branch and the validation branch), and the existing `obsidianScheme` constant style.
- `pkg/handler/obsidian-redirect_test.go` — the Ginkgo/Gomega suite. Note the `request(rawQuery string) *httptest.ResponseRecorder` helper and the existing `Context` blocks: valid requests, rejections, open-redirect falsifier, allowlist parsing, and the file-allowlist boundary.
- `pkg/handler/handler_suite_test.go` — Ginkgo bootstrap; tests run through `RunSpecs`.
- `docs/dod.md` — project definition of done.
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md` — Ginkgo/Gomega conventions.
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-http-service-guide.md` — handler conventions.

Do not change any file outside `pkg/handler/obsidian-redirect.go` and `pkg/handler/obsidian-redirect_test.go`.

Purging any `/obsidian` response that a shared cache already stored before this change is an operator step (see the spec's Non-goals and Failure Modes) and is out of scope for this code change.
</context>

<requirements>
1. In `pkg/handler/obsidian-redirect.go`, add a package-level constant next to the existing `obsidianScheme` constant:

   ```go
   // cacheControlNoStore is the directive every /obsidian response carries. The
   // redirect target is derived from an allowlist decision made for one request;
   // a shared cache that replays the answer hands a caller a target this service
   // never validated for that request, and a cached response survives an
   // allowlist change. `no-store` tells every intermediary not to retain it.
   const cacheControlNoStore = "no-store"
   ```

   Use the literal header name `"Cache-Control"` at the call sites; `net/http` does not define a constant for it.

2. In the success branch of the `http.HandlerFunc` inside `NewObsidianRedirectHandler`, set the header after `Location` and before the status write. The resulting order must be:

   ```go
   resp.Header().Set("Location", target)
   resp.Header().Set("Cache-Control", cacheControlNoStore)
   resp.WriteHeader(http.StatusFound)
   ```

   Keep `http.Redirect` unused and keep `target` written verbatim so `Location` stays byte-for-byte the encoded deeplink.

3. In `writeRejection`, set the header before the status write:

   ```go
   func writeRejection(resp http.ResponseWriter, message string) {
       resp.Header().Set("Cache-Control", cacheControlNoStore)
       resp.WriteHeader(http.StatusBadRequest)
       if _, err := resp.Write([]byte(message)); err != nil {
           glog.V(2).Infof("write rejection response failed: %v", err)
       }
   }
   ```

   `writeRejection` is shared by the `url`-parameter rejection and the validation rejection, so both rejection branches inherit the header. Do not change the rejection body text. Add a short comment noting that any future rejection branch must route through `writeRejection` so it inherits the directive.

4. In `pkg/handler/obsidian-redirect_test.go`, assert the header by reading the committed response snapshot, `resp.Result().Header`, NOT the live `resp.Header()` map. `httptest.ResponseRecorder.Result()` snapshots the header at the moment `WriteHeader` is called, so an assertion on `resp.Result().Header` fails if the `Set` happens after the status is committed, while `resp.Header()` would still show it. Use:

   ```go
   Expect(resp.Result().Header.Get("Cache-Control")).To(Equal("no-store"))
   ```

5. Add a dedicated Ginkgo example in the `Context("a valid request")` block, for example `It("marks the redirect as non-cacheable")`, that issues a valid request and asserts `resp.Result().Header.Get("Cache-Control")` equals `"no-store"`.

6. Add the same assertion to every rejection example. There are exactly these existing rejection `It`s in `Context("rejections")`:
   - `rejects a vault outside the allowlist with 400`
   - `rejects a file outside the path allowlist with 400`
   - `rejects a file that traverses upward with 400`
   - `rejects an absolute file with 400`
   - `rejects a file containing CRLF, which would otherwise reach the Location header`
   - `rejects a vault containing CRLF`
   - `rejects a missing vault with 400`
   - `rejects a missing file with 400`
   - `rejects a url parameter outright with 400`

   Also add it to the boundary rejection `rejects a sibling directory whose name shares the prefix` in `Context("the file allowlist boundary")`, and inside the loop of `returns a rejection status for every hostile input` in `Context("open-redirect falsifier")` so each hostile query is checked as well. In the falsifier loop, capture the recorder once per query (e.g. `resp := request(query)`) and assert both the status and the header on that same recorder.

7. Do not remove, weaken, or rewrite any existing `Location` assertion. Lines containing `obsidian://open` in the test file must remain unchanged. The only test changes are added assertions and, if needed, the one new valid-request example.

8. Verify the regression lock manually as a self-check: temporarily remove the `resp.Header().Set("Cache-Control", cacheControlNoStore)` line from the success branch, run `go test ./pkg/handler/...`, and confirm it fails; then restore the line exactly and confirm the suite passes. The final committed state MUST include both `Set` calls.

9. Leave the admin handlers and their routes untouched. This change is scoped to the `/obsidian` handler only.
</requirements>

<constraints>
- The emitted `Location` stays byte-for-byte the encoded deeplink; `http.Redirect` is still not used.
- No new dependency.
- The existing rejection body text is unchanged.
- Existing tests keep passing without modification beyond adding the new assertions.
- Do not change the allowlist logic, the accepted parameters, or the emitted deeplink format.
- Do not add a cache directive to `/healthz`, `/readiness`, `/metrics`, `/setloglevel`, `/gc`, `/testloglevel`, or `/sentryalert`.
- Do not add or remove any other response header.
- Do not introduce any caching layer, positive or negative.
- Do not change the status codes (`302` for a redirect, `400` for a rejection).
- Do NOT commit — dark-factory handles git.
- Existing tests must still pass.
- Follow the existing Ginkgo/Gomega conventions and the `docs/dod.md` definition of done.
</constraints>

<verification>
Run `go test ./pkg/handler/...` — must pass.
Run `make precommit` — must pass.
</verification>
