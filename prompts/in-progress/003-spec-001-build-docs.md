---
status: approved
spec: [001-cluster-build-arm]
created: "2026-11-09T17:55:00Z"
queued: "2026-11-09T17:11:18Z"
branch: dark-factory/cluster-build-arm
---

# Document the two build arms in README and CHANGELOG

<summary>
- The README now documents both build paths and when to use each.
- Readers learn the local build is the default and keeps tagging by branch.
- Readers learn how to trigger an unattended cluster build from a committed ref.
- Readers learn the one-time bootstrap step required before a cluster's first build.
- The changelog records the new capability as an unreleased change.
- No code or build behaviour changes in this prompt.
</summary>

<objective>
Make the documented build path match reality: add a `## Build` section to `README.md` describing the local (default) and cluster build arms, and add a `## Unreleased` section to `CHANGELOG.md` with one bullet for this change. `README.md` is the in-repo home for the two-arm topology because the spec is removed after implementation.
</objective>

<context>
Read these before editing:
- `README.md` — currently has `## Run locally`, `## Endpoint`, `## Deploy`; add a `## Build` section.
- `CHANGELOG.md` — the SemVer preamble is followed by version sections; add `## Unreleased` immediately after the preamble and before `## v0.1.12`.
- `Makefile.docker` — the source of truth for the invocation and defaults documented below.
- `hack/build-cluster-bootstrap.sh` — the one-time cluster bootstrap; it takes the cluster alias or kubectl context as its first argument.
- `docs/dod.md` — Definition of Done.
- `/home/node/.claude/plugins/marketplaces/coding/docs/changelog-guide.md` — changelog entry conventions (conventional prefixes, `## Unreleased` placement).
</context>

<requirements>
1. Add a `## Build` section to `README.md` (the heading must be exactly `## Build`). It must state all of the following:
   - `make build` is the **default** and builds the image locally, tagging `docker.io/bborbe/obsidian-redirector:<branch>` (the checked-out branch with `/` replaced by `-`).
   - The local build runs through `docker buildx build` with `rewrite-timestamp` clamped to the built commit, so two builds of one commit produce the same image.
   - The exact cluster invocation, verbatim: `make build BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>`.
   - The cluster arm renders a Kubernetes Job, waits for it, and pushes `docker.io/bborbe/obsidian-redirector:<tag>` from inside the cluster.
   - `VERSION` has no default: the build refuses unless the caller supplies it.
   - `REF` must be a full 40-character commit SHA that exists on `origin`, because the Job clones the repository at that ref.
   - `buca` is a deploy command (`make buca` builds locally, pushes, cleans the local image, and applies the manifests). For a cluster build, `make buca BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>` builds and pushes from inside the cluster, and the local push/cleanup steps are skipped.
   - `hack/build-cluster-bootstrap.sh` must be run once against a cluster before its first build. It is idempotent and creates the `build` namespace and the `buildkit-push` secret. Show the command `hack/build-cluster-bootstrap.sh <cluster>`.

   Suggested content (adapt wording, keep the exact tokens the verification greps for):
   ````markdown
   ## Build

   `make build` builds the image locally and is the default. It tags
   `docker.io/bborbe/obsidian-redirector:<branch>` (the checked-out branch with `/`
   replaced by `-`). The local build runs through `docker buildx build` with
   `rewrite-timestamp` clamped to the built commit, so two builds of one commit
   produce the same image.

   To build inside a Kubernetes cluster instead — unattended, from a committed ref —
   pass a cluster alias or kubectl context and a tag:

   ```bash
   make build BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>
   ```

   The cluster arm renders a Job, waits for it, and pushes
   `docker.io/bborbe/obsidian-redirector:<tag>` from inside the cluster. `VERSION`
   has no default: the build refuses unless you pass it. `REF` must be a full
   40-character commit SHA that exists on `origin`, because the Job clones the
   repository at that ref.

   `buca` is the deploy command: `make buca` builds locally, pushes, cleans the
   local image, and applies the manifests. For a cluster build use
   `make buca BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>`; the local push and
   cleanup steps are skipped because the image is pushed from inside the cluster.

   Before the first cluster build against a cluster, run the bootstrap once (it is
   idempotent; it creates the `build` namespace and the `buildkit-push` secret):

   ```bash
   hack/build-cluster-bootstrap.sh <cluster>
   ```
   ````

2. Add a `## Unreleased` section to `CHANGELOG.md`, immediately after the SemVer preamble and before the first `## vX.Y.Z` section. It must contain exactly one bullet for this change, using a conventional prefix:
   ```markdown
   ## Unreleased

   - feat: add a cluster build arm so the image can be built unattended from any committed ref, and move the local arm to `docker buildx build` so two builds of one commit produce the same image
   ```

3. Do not change the existing README sections or the existing changelog entries. Do not move, edit, or delete the changelog preamble.

4. Acceptance criteria covered by this prompt:
   - `grep -c '^## Build$' README.md` returns 1.
   - `grep -c 'BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>' README.md` returns at least 1.
   - `grep -c 'build-cluster-bootstrap.sh' README.md` returns at least 1.
   - `grep -c '^## Unreleased$' CHANGELOG.md` returns 1.
</requirements>

<constraints>
- `README.md` is the in-repo home for the two-arm topology; the spec dies after implementation, the README does not.
- The documented invocation must match `Makefile.docker` exactly: `make build BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>`.
- The changelog entry goes under `## Unreleased`, after the SemVer preamble and before the newest version section.
- Do NOT bump the nuke mirror pin (`VERSION := v0.1.2`) or the tag in `obsidian-redirector-sts.yaml`; promoting an image to the clusters is a separate deploy step.
- Do not touch `.gitignore` or any Go source.
- Do NOT commit — dark-factory handles git.
- Existing tests must still pass.
</constraints>

<verification>
Run from the repo root. All commands are container-executable.

```bash
set -e
grep -c '^## Build$' README.md
grep -c 'BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>' README.md
grep -c 'build-cluster-bootstrap.sh' README.md
grep -c '^## Unreleased$' CHANGELOG.md
grep -q 'docker buildx build' README.md
grep -q 'buca' README.md
make precommit
```

Before finishing, re-run the block above and confirm it passes; walk each acceptance criterion in requirement 4 against the change.
</verification>
