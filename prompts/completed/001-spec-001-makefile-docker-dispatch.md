---
status: completed
spec: [001-cluster-build-arm]
summary: Rewrote Makefile.docker to dispatch make build between a reproducible local buildx arm and a cluster Job arm via BUILD_TARGET, and added the CHANGELOG Unreleased entry
execution_id: 5f72f751-9f28-5382-8ed4-54fa711781b0-exec-001-spec-001-makefile-docker-dispatch
dark-factory-version: dev
created: "2026-11-09T17:55:00Z"
queued: "2026-11-09T17:01:39Z"
started: "2026-11-09T17:01:41Z"
completed: "2026-11-09T17:11:42Z"
branch: dark-factory/cluster-build-arm
---

# Makefile.docker build dispatch: local buildx arm and cluster Job arm

<summary>
- `make build` can now run either on this machine or inside a Kubernetes cluster.
- The default stays the local build and keeps tagging by branch, exactly as today.
- A cluster build refuses to start unless the caller supplied a tag.
- Both build paths publish to the same Docker Hub repository, differing only in tag.
- Local push and cleanup steps become no-ops when the build target is a cluster.
- The local build now goes through buildx with reproducible timestamp flags.
- Two builds of one commit now produce the same image.
- No new branch guard is added, so a scratch-tag build still works.
</summary>

<objective>
Teach `Makefile.docker` to dispatch `make build` between the existing local arm and a new cluster arm selected by `BUILD_TARGET`, and make the local arm reproducible via `docker buildx build` with `rewrite-timestamp` clamped to the built commit. The local arm must keep its current command shape and its current tag `$(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH)`; the cluster arm must push `$(DOCKER_REGISTRY)/$(SERVICE):$(VERSION)` from inside the cluster and refuse to run without a caller-supplied `VERSION`.
</objective>

<context>
Read these before editing:
- `Makefile` — includes `Makefile.variables`, `Makefile.precommit`, `Makefile.docker`, `example.env`; `SERVICE = bborbe/obsidian-redirector` is defined here.
- `Makefile.docker` — the current single local recipe; this is the file you rewrite.
- `Makefile.variables` — `BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD | tr '/' '-')`.
- `example.env` — `export DOCKER_REGISTRY=docker.io` and `export IMAGE=bborbe/obsidian-redirector` (unqualified).
- `Dockerfile` — consumes `DOCKER_REGISTRY`, `BRANCH`, `BUILD_GIT_VERSION`, `BUILD_GIT_COMMIT`, `BUILD_DATE` as build args.
- `docs/dod.md` — Definition of Done.
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-makefile-commands.md` — Makefile conventions.

Optional provenance reference (the pattern this dispatch mirrors, read-only):
`https://raw.githubusercontent.com/bborbe/go-skeleton/eb050aa20bb6194d8af7e47baf8880df789bd1ca/Makefile.docker`.
</context>

<requirements>
1. Rewrite `Makefile.docker` to the shape below. Keep the existing `DIRS`, `apply`, and `buca` targets unchanged. The exact text matters — later prompts and the spec's verification grep for it.

2. Add the build-selection variables near the top of the file:
   - `BUILD_TARGET ?= local`
   - `BUILD_NAMESPACE ?= build`
   - `BUILD_TIMEOUT ?= 3600`
   Do NOT introduce a `BUILD_DOCKERFILE` variable. The Non-goals forbid extra build knobs beyond `BUILD_TARGET`, `VERSION`, `BUILD_NAMESPACE`, `BUILD_TIMEOUT` and the repository URL; the vendored runner already defaults `DOCKERFILE` to `Dockerfile`.

3. Add the repository and push-secret values as literal `:=` assignments, NOT `?=`:
   - `REPO_URL := https://github.com/bborbe/obsidian-redirector.git`
   - `PUSH_SECRET := buildkit-push`
   Do NOT define `BUILD_REPO_URL` or `BUILD_PUSH_SECRET` anywhere. The copied runner's `?=` defaults are overridable from the command line; a caller could then point a push-credential-bearing Job at an arbitrary repository. The literal assignment is the security control. The verification greps for `^REPO_URL *:=` to be present and `BUILD_REPO_URL ?=` to be absent.

4. Add the build clock variables:
   - `SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct)`
   - `BUILD_DATE ?= $(shell git log -1 --format=%cI)`
   `SOURCE_DATE_EPOCH` is what `rewrite-timestamp` clamps layer and image timestamps to; without it the reproducibility acceptance criterion cannot pass.

5. Rewrite the `build` target as a make conditional on `$(BUILD_TARGET)`:

   **Local branch (`$(BUILD_TARGET)` is `local`)** — replace the current recipe:
   - Remove `DOCKER_BUILDKIT=1` and `--rm=true`.
   - Change `docker build` to `docker buildx build`.
   - Add `--provenance=false` and `--sbom=false`.
   - Add `--build-arg SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH)`.
   - **Replace the inline wall clock** `--build-arg BUILD_DATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ)` with `--build-arg BUILD_DATE=$(BUILD_DATE)`. Leaving the old line keeps a per-run label and fails the reproducibility criterion.
   - End the recipe with `--output type=docker,rewrite-timestamp=true,unpack=false`.
   - Keep the tag exactly `$(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH)`, `-f Dockerfile .`, and `--platform=linux/amd64`.
   - Keep `--build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY)`, `--build-arg BRANCH=$(BRANCH)`, `--build-arg BUILD_GIT_VERSION=$$(git describe --tags --always --dirty)`, `--build-arg BUILD_GIT_COMMIT=$$(git rev-parse --short HEAD)`.

   **Cluster branch (`$(BUILD_TARGET)` is anything else)** — render the Job through the vendored runner by exporting the runner's inputs and invoking it on a line of its own:
   - `BUILD_TARGET='$(BUILD_TARGET)'`
   - `NAMESPACE='$(BUILD_NAMESPACE)'`
   - `REPO_URL='$(REPO_URL)'`
   - `REF='$(REF)'`
   - `TAG='$(VERSION)'`
   - `IMAGE='$(DOCKER_REGISTRY)/$(SERVICE)'` — the **composed** value. Never pass `example.env`'s bare `IMAGE=bborbe/obsidian-redirector`; the runner would push to an unprefixed path with no registry.
   - `PUSH='true'`
   - `PUSH_SECRET='$(PUSH_SECRET)'`
   - `TIMEOUT='$(BUILD_TIMEOUT)'`
   - the invocation line must begin (after leading whitespace) with `hack/build-cluster.sh`, because the dispatch passes the runner's inputs as environment assignments on the preceding continuation lines.

6. Add the `VERSION` refusal **inside the cluster branch, before the runner line**, as a parse-time make error:
   ```make
   ifeq ($(filter file undefined,$(origin VERSION)),$(origin VERSION))
   $(error VERSION is required for a cluster build; pass VERSION=<tag> on the make command line)
   endif
   ```
   This fires when `$(origin VERSION)` is `file` **or** `undefined`. `VERSION` is defined nowhere in this repo, so a guard that tested only for `file` would never fire. The error fires at parse time, before any Job is rendered or applied. Do NOT add a default for `VERSION`.

7. Do NOT add a `check-version-tag` prerequisite to `build:`. This repo has no such target and no `Makefile.env`; adding one breaks the scratch-tag build outright.

8. Guard `upload` and `clean` with the same `ifeq ($(BUILD_TARGET),local)` conditional so that, for a non-local target, both expand to nothing. `make buca BUILD_TARGET=<cluster>` must not attempt a local `docker push` or `docker rmi` of an image the cluster arm never loaded locally. Leave `apply` and the `buca: build upload clean apply` dependency list unchanged.

9. Reference shape for the rewritten `Makefile.docker` (copy this structure; keep the existing `DIRS`/`apply`/`buca` bodies):
   ```make
   DIRS += $(shell find */* -maxdepth 0 -name Makefile -exec dirname "{}" \;)

   # BUILD_TARGET selects where `make build` runs. `local` (the default) uses the
   # local Docker daemon; any other value must name a kubectl context and builds
   # inside that cluster as a Job rendered from hack/build-job.yaml.
   BUILD_TARGET ?= local
   BUILD_NAMESPACE ?= build
   BUILD_TIMEOUT ?= 3600

   # Assigned with `:=` (literal), never `?=`: these flow into a Job that mounts
   # a push credential, and a command-line override would let a caller point that
   # credential-bearing Job at an arbitrary repository.
   REPO_URL := https://github.com/bborbe/obsidian-redirector.git
   PUSH_SECRET := buildkit-push

   # Build clock pinned to the built commit instead of the wall clock, so two
   # builds of the same commit can produce the same image.
   SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct)
   BUILD_DATE ?= $(shell git log -1 --format=%cI)

   .PHONY: build
   build:
   ifeq ($(BUILD_TARGET),local)
   	docker buildx build \
   	--platform=linux/amd64 \
   	--provenance=false \
   	--sbom=false \
   	--build-arg DOCKER_REGISTRY=$(DOCKER_REGISTRY) \
   	--build-arg BRANCH=$(BRANCH) \
   	--build-arg BUILD_GIT_VERSION=$$(git describe --tags --always --dirty) \
   	--build-arg BUILD_GIT_COMMIT=$$(git rev-parse --short HEAD) \
   	--build-arg BUILD_DATE=$(BUILD_DATE) \
   	--build-arg SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH) \
   	-t $(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH) \
   	-f Dockerfile . \
   	--output type=docker,rewrite-timestamp=true,unpack=false
   else
   ifeq ($(filter file undefined,$(origin VERSION)),$(origin VERSION))
   $(error VERSION is required for a cluster build; pass VERSION=<tag> on the make command line)
   endif
   	BUILD_TARGET='$(BUILD_TARGET)' \
   	NAMESPACE='$(BUILD_NAMESPACE)' \
   	REPO_URL='$(REPO_URL)' \
   	REF='$(REF)' \
   	TAG='$(VERSION)' \
   	IMAGE='$(DOCKER_REGISTRY)/$(SERVICE)' \
   	PUSH='true' \
   	PUSH_SECRET='$(PUSH_SECRET)' \
   	TIMEOUT='$(BUILD_TIMEOUT)' \
   	hack/build-cluster.sh
   endif

   .PHONY: upload
   upload:
   ifeq ($(BUILD_TARGET),local)
   	docker push $(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH)
   endif

   .PHONY: clean
   clean:
   ifeq ($(BUILD_TARGET),local)
   	docker rmi $(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH) || true
   	docker builder prune --max-used-space 21474836480 -f || true
   endif

   .PHONY: apply
   apply:
   	@for i in $(DIRS); do \
   		cd $$i; \
   		echo "apply $${i}"; \
   		make apply; \
   		cd ..; \
   	done

   .PHONY: buca
   buca: build upload clean apply
   ```
   The `hack/build-cluster.sh` file is added by the next prompt; this prompt's `make -n` checks pass without it because `make -n` only prints the command.

10. Acceptance criteria covered by this prompt (container-verifiable):
    - `BUILD_TARGET` defaults to `local`; `make -n build` prints a `docker buildx build` recipe carrying `--output type=docker,rewrite-timestamp=true,unpack=false`, `--provenance=false`, `--sbom=false`, and no plain `docker build ` invocation.
    - `make -n build BUILD_TARGET=nukeprod REF=<sha> VERSION=probe` prints `IMAGE='docker.io/bborbe/obsidian-redirector'` and `TAG='probe'`.
    - `make -n build BUILD_TARGET=nukeprod REF=<sha>` with no `VERSION` exits non-zero and prints a message naming `VERSION`.
    - `make -n buca BUILD_TARGET=nukeprod REF=<sha> VERSION=probe` prints neither `docker push` nor `docker rmi`; `make -n buca` still prints both.
    - `REPO_URL` is assigned with `:=`; `BUILD_REPO_URL ?=` is absent.

    The real image build and the real cluster build are host-side checks recorded on the spec's Verification section; they are not part of this prompt's container verification.
</requirements>

<constraints>
- The local arm's tag must not change: `$(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH)`.
- The cluster arm passes the runner's required `IMAGE` input explicitly at the call site as `IMAGE='$(DOCKER_REGISTRY)/$(SERVICE)'`. `example.env` also exports `IMAGE=bborbe/obsidian-redirector`, which is unqualified — a dispatch that let that value reach the runner would push to `bborbe/obsidian-redirector` with no registry. The explicit assignment is what overrides it.
- `VERSION` has no definition anywhere in this repo, so the refusal must fire when `$(origin VERSION)` is `file` **or** `undefined`. A guard testing only for `file` never fires here.
- No `check-version-tag` prerequisite on `build:`.
- `SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct)` must be defined and passed as `--build-arg SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH)` on the local arm, alongside `BUILD_DATE ?= $(shell git log -1 --format=%cI)`. The old inline `--build-arg BUILD_DATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ)` must be replaced by `--build-arg BUILD_DATE=$(BUILD_DATE)`.
- `REPO_URL` and `PUSH_SECRET` are assigned with `:=` (a literal), never inherited from `BUILD_REPO_URL ?=` / `BUILD_PUSH_SECRET ?=`.
- The Job template passes `DOCKER_REGISTRY=docker.io` as a build argument; do not change that to a private mirror.
- No new scenario. The behaviour is reachable by checks on the rendered invocation plus one real cluster build.
- Do NOT commit — dark-factory handles git.
- Existing tests must still pass; do not modify Go source.
</constraints>

<verification>
Run from the repo root. All commands are container-executable.

```bash
set -e
make precommit
grep -c '^BUILD_TARGET' Makefile.docker
grep -cE '^[[:space:]]*hack/build-cluster\.sh' Makefile.docker
grep -n '^REPO_URL *:=' Makefile.docker
! grep -q 'BUILD_REPO_URL ?=' Makefile.docker
! grep -q 'BUILD_PUSH_SECRET ?=' Makefile.docker
grep -c '^SOURCE_DATE_EPOCH' Makefile.docker
grep -c '^BUILD_DATE' Makefile.docker
make -n build | grep -q 'docker buildx build'
make -n build | grep -q -- '--output type=docker,rewrite-timestamp=true,unpack=false'
make -n build | grep -q -- '--provenance=false'
make -n build | grep -q -- '--sbom=false'
make -n build | grep -q -- '--build-arg SOURCE_DATE_EPOCH='
make -n build | grep -q -- '--build-arg BUILD_DATE='
! make -n build | grep -q 'date -u +%Y-%m-%dT%H:%M:%SZ'
! make -n build | grep -qE 'docker build[[:space:]]+--'
make -n build BUILD_TARGET=nukeprod REF=0000000000000000000000000000000000000000 VERSION=probe | grep -q "IMAGE='docker.io/bborbe/obsidian-redirector'"
make -n build BUILD_TARGET=nukeprod REF=0000000000000000000000000000000000000000 VERSION=probe | grep -q "TAG='probe'"
! make -n buca BUILD_TARGET=nukeprod REF=0000000000000000000000000000000000000000 VERSION=probe | grep -q 'docker push'
! make -n buca BUILD_TARGET=nukeprod REF=0000000000000000000000000000000000000000 VERSION=probe | grep -q 'docker rmi'
make -n buca | grep -q 'docker push'
make -n buca | grep -q 'docker rmi'
if make -n build BUILD_TARGET=nukeprod REF=0000000000000000000000000000000000000000 >/dev/null 2>&1; then echo "expected VERSION refusal"; exit 1; fi
make -n build BUILD_TARGET=nukeprod REF=0000000000000000000000000000000000000000 2>&1 | grep -q 'VERSION'
```

Before finishing, re-run the block above and confirm it passes; walk each acceptance criterion in requirement 10 against the change.
</verification>

<!-- OPEN QUESTION (auditor/reviewer): the spec does not state the literal `REPO_URL` or `PUSH_SECRET` values. This prompt assumes `REPO_URL := https://github.com/bborbe/obsidian-redirector.git` (the repo's GitHub URL) and `PUSH_SECRET := buildkit-push` (named in the spec's Assumptions section). Correct these two literals if the operator expects different values. -->
<!-- OPEN QUESTION (auditor/reviewer): the `$(error)` guard is evaluated at make parse time, so ANY non-local invocation without VERSION errors — including a non-`build` target such as `make clean BUILD_TARGET=nukeprod`. This is the strictest reading of "refuses before any Job is created". If a narrower scope is wanted, the guard must move to an invocation-time check instead; the spec's AC only exercises `make build`. -->
