# `hack/` — vendored cluster build runner

These five files are **copied unmodified** from two upstream repos, pinned to a
full commit SHA. Do not edit them here: a local fix forks the runner and the next
copy silently reverts it. Fix upstream, then re-copy and bump the pin below.

| File | Source repo | Pinned commit |
|---|---|---|
| `build-cluster.sh` | `bborbe/agent-task-executor` | `d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40` |
| `cluster-alias.sh` | `bborbe/agent-task-executor` | `d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40` |
| `build-job.yaml` | `bborbe/agent-task-executor` | `d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40` |
| `build-cluster-bootstrap.sh` | `bborbe/go-skeleton` | `eb050aa20bb6194d8af7e47baf8880df789bd1ca` |
| `dockerconfig.json.tpl` | `bborbe/go-skeleton` | `eb050aa20bb6194d8af7e47baf8880df789bd1ca` |

`agent-task-executor`'s `build-job.yaml` carries `automountServiceAccountToken:
false`; `go-skeleton`'s does not, which is why the split is three-plus-two rather
than a single source.

## Known upstream limitations, accepted here

Recorded so the next reader does not rediscover them, and so a future copy can
check whether upstream has fixed them.

- **`BUILD_NAMESPACE` is ignored by the Job.** `Makefile.docker` exposes
  `BUILD_NAMESPACE ?= build` and passes it to `build-cluster.sh`, but
  `build-job.yaml:31` hardcodes `namespace: build` and has no `__NAMESPACE__`
  token to substitute. The knob therefore only takes effect through the
  `kubectl` calls, not in the rendered manifest. With the default value the
  behaviour is correct; setting a different namespace would split the two.
- **The wait loop swallows `kubectl` read errors.** `build-cluster.sh` treats a
  failed `get pod` as "not ready yet" rather than a hard error, so a lost
  cluster connection looks like a slow build until the timeout expires.
- **Job names can collide.** The name is derived from the repository, ref and
  tag; two concurrent builds of the same (repo, ref, tag) target the same Job,
  and the second is refused with the existing Job named.
- **The bootstrap interpolates the credential into JSON without escaping.**
  `build-cluster-bootstrap.sh` renders the dockerconfigjson with `printf`
  rather than a JSON encoder, so a password containing `"` or `\` produces
  malformed JSON. The failure is loud — `kubectl create secret` rejects it —
  and the credential is never echoed or written to the repo, so the exposure is
  a failed bootstrap, not a leak.

## Updating the pin

```bash
git -C ~/Documents/workspaces/agent-task-executor fetch
git -C ~/Documents/workspaces/go-skeleton fetch
# re-copy the five files at the new SHAs, then update this table and CHANGELOG.md
```

Verify byte-identity before committing:

```bash
git -C ~/Documents/workspaces/agent-task-executor show <sha>:hack/build-cluster.sh \
  | diff - hack/build-cluster.sh
```
