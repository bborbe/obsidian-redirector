---
status: completed
approved: "2026-11-09T16:24:01Z"
generating: "2026-11-09T16:39:52Z"
prompted: "2026-11-09T16:53:29Z"
verifying: "2026-11-09T17:13:47Z"
completed: "2026-11-09T17:13:54Z"
branch: feat/cluster-build-arm
---

## Summary

- This repo's image can currently be produced on one machine only: the Mac.
- Its sibling repos can also build their image inside a Kubernetes cluster, unattended, from any committed ref.
- Add that second build path here as an arm beside the existing local one, leaving the local tag as it is.
- The cluster arm builds a named commit as a Kubernetes Job and pushes the result to `docker.io/bborbe/obsidian-redirector` — the repository this repo already tags today.
- The local arm moves to `docker buildx build` with the reproducibility flags, which is what `go-skeleton/docs/build-migration-guide.md` § 1 requires of the local arm of every migrated repo — plain `docker build` pins the image config clock but not the layer tars, so two builds of one commit still differ.

## Problem

`obsidian-redirector` runs on nukedev and nukeprod, and its image can only be cut on this Mac. `Makefile.docker` carries one recipe: `docker build` against the local daemon, tagging `$(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH)`. There is no `BUILD_TARGET`, no `hack/`, and no way to produce the image from a committed ref without a human at the keyboard.

The deployment consumes the image through nuke's mirror step: `nuke@prod:obsidian-redirector/Makefile:15` pins `VERSION := v0.1.2` and runs a hand-rolled `mirror:` target that pulls from Docker Hub, retags, and pushes into `$(REGISTRY_HOST):443/bborbe/obsidian-redirector:$(VERSION)`; `obsidian-redirector-sts.yaml:86` pulls that tag. So the Docker Hub repository is the publish surface this repo already depends on, and a cluster arm that pushes there needs no change to the deploy path.

The local arm is not reproducible today, and the migration is what fixes it: `go-skeleton/docs/build-migration-guide.md` § 1 requires `docker buildx build` with `--output type=docker,rewrite-timestamp=true,unpack=false` and `--provenance=false --sbom=false`, because `rewrite-timestamp` is rejected together with `unpack`, and the attestation manifests carry their own timestamps so they change the image ID even when manifest and config match. A repo that keeps plain `docker build` on the local arm still produces a different image per run.

Three properties of this repo make a naive copy from a sibling fail at review rather than at runtime:

- **`SERVICE` and `IMAGE` both exist, and the dispatch must choose one deliberately.** `Makefile:6` is `SERVICE = bborbe/obsidian-redirector` and `example.env:4` is `export IMAGE=bborbe/obsidian-redirector` — the latter unqualified, with no registry. `example.env:2` sets `DOCKER_REGISTRY=docker.io`. So `$(DOCKER_REGISTRY)/$(SERVICE)` composes `docker.io/bborbe/obsidian-redirector`, and so does `$(DOCKER_REGISTRY)/$(IMAGE)`; both land on the right repository. The copied runner hard-requires `IMAGE` (`agent-task-executor/hack/build-cluster.sh:41` `require IMAGE`), so the dispatch must pass it explicitly at the call site; it must pass the *composed* value, never `example.env`'s bare `bborbe/obsidian-redirector`, which the runner would push to as an unprefixed path.
- **`VERSION` is defined nowhere.** It appears in no `Makefile`, no `Makefile.*`, and not in `example.env`. A guard that refuses only when `$(origin VERSION)` is `file` never fires in this repo; the caller would instead get the runner's generic `TAG is required` (`hack/build-cluster.sh:40`) from inside the runner, not a refusal naming `VERSION` before the Job is rendered.
- **There is no branch guard to narrow.** This repo has no `Makefile.env` and no `check-version-tag`; `build` runs on any branch. A copied `check-version-tag` prerequisite would break the scratch-tag build outright.

## Goal

`make build` on this repo produces the image locally, tagging the same reference it tags today, and `make build BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>` produces it inside that cluster as a Kubernetes Job, pushing `docker.io/bborbe/obsidian-redirector:<tag>`. Both arms work from the same checkout, and the cluster arm works from a fresh clone with no local preparation. The local arm's *mechanism* changes to `docker buildx build` so that two builds of one commit produce the same image; its command and its tag do not.

## Non-goals

- Renaming the image or changing the repository it publishes to. `docker.io/bborbe/obsidian-redirector` is what the mirror step pulls and what this spec keeps.
- Bumping the nuke mirror pin (`VERSION := v0.1.2`) or the tag in `obsidian-redirector-sts.yaml`. Promoting a new image to the clusters is a separate deploy step.
- Building at release or tag time. This spec adds the capability; *when* a build is triggered is a separate concern.
- Multi-arch images. `linux/amd64` only, matching the cluster nodes.
- Extracting the shared build runner into a common upstream template. The files are copied, not shared.
- Extra build knobs beyond `BUILD_TARGET`, `VERSION`, `BUILD_NAMESPACE`, `BUILD_TIMEOUT` and the repository URL the template carries.
- Changing the local arm's tag. It stays `$(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH)`.

## Acceptance Criteria

- [ ] **The local arm still builds, and still tags by branch** — with the tag removed first (`DOCKER_CONTEXT=orbstack docker rmi docker.io/bborbe/obsidian-redirector:<branch> || true`), `DOCKER_CONTEXT=orbstack docker image inspect docker.io/bborbe/obsidian-redirector:<branch>` fails before the run and succeeds after `DOCKER_CONTEXT=orbstack make build` exits 0, and the image's `.Created` renders the built commit's epoch in UTC rather than the wall clock — `docker image inspect --format '{{.Created}}'` begins with `$(date -u -r "$(git log -1 --format=%ct)" +%Y-%m-%dT%H:%M:%S)`. `<branch>` is the checked-out branch with `/` replaced by `-`. Evidence: the pre-run failure text, the exit code, the exact reference, the image `.Id`, and the two UTC timestamps side by side. ⚠️ Compare **epochs rendered in UTC**, not `%cI` strings: `%cI` carries the local offset (`…+01:00`) while Docker renders `.Created` as UTC (`…Z`), so a string comparison of the two false-fails on a correct build. ⚠️ `.Created` must **not** be asserted as later than the run start: `rewrite-timestamp` rewrites it to `SOURCE_DATE_EPOCH`, which is the built commit's epoch and therefore always earlier.
- [ ] **The local arm runs through buildx with the reproducibility flags, and two builds of one commit agree** — `make -n build` prints a `docker buildx build` recipe line carrying `--output type=docker,rewrite-timestamp=true,unpack=false`, `--provenance=false` and `--sbom=false`, and prints no recipe line invoking `docker build ` followed by a flag (the plain builder, as distinct from `docker buildx build`); and with the tag removed between runs (`DOCKER_CONTEXT=orbstack docker rmi docker.io/bborbe/obsidian-redirector:<branch> || true`), two consecutive `DOCKER_CONTEXT=orbstack make build` runs report the same `docker image inspect --format '{{.Id}}' docker.io/bborbe/obsidian-redirector:<branch>`. Evidence: the printed recipe line, and the two image IDs side by side.
- [ ] **Post-Deploy (Rung-3):** the cluster arm builds and pushes from a fresh clone — `make build BUILD_TARGET=nukeprod REF=<full 40-character sha> VERSION=<scratch-tag>` exits 0, the Job name and the build pod's `spec.nodeName` are recorded, `crane digest docker.io/bborbe/obsidian-redirector:<scratch-tag>` resolves, and `docker image inspect` on that same reference fails on this Mac with `No such image`. Evidence: one row carrying the cluster name, the scratch tag, the exit code, the Job name, the node, the resolved digest and the inspect failure text.
  - `deploy_check:` `kubectlnukeprod config view --minify -o jsonpath='{.clusters[0].name}'`
  - `deploy_target:` `nuke-prod`
- [ ] **Post-Deploy (Rung-3):** the cluster arm composes the Docker Hub reference the deployment's mirror pulls — `make -n build BUILD_TARGET=nukeprod REF=<sha> VERSION=probe` prints `IMAGE='docker.io/bborbe/obsidian-redirector'` and `TAG='probe'` as two assignments, and `crane digest docker.io/bborbe/obsidian-redirector:<scratch-tag>` equals the `containerimage.digest` the build Job itself reports for that same run. Evidence: the two `make -n` assignment lines, plus the two digest values side by side.
  - `deploy_check:` `kubectlnukeprod config view --minify -o jsonpath='{.clusters[0].name}'`
  - `deploy_target:` `nuke-prod`
- [ ] **Post-Deploy (Rung-3):** the cluster arm refuses a tag it did not receive from the caller — `make build BUILD_TARGET=nukeprod REF=<sha>` with no `VERSION` on the command line exits non-zero and prints a message naming `VERSION`, before any Job is applied. Evidence: the exit code, the message, and `kubectlnukeprod -n build get jobs` showing no new Job for that run.
  - `deploy_check:` `kubectlnukeprod config view --minify -o jsonpath='{.clusters[0].name}'`
  - `deploy_target:` `nuke-prod`
- [ ] **`hack/` holds all five files and the cluster branch invokes the runner** — at the recorded head SHA, `git ls-tree --name-only HEAD hack/` lists `build-cluster.sh`, `build-cluster-bootstrap.sh`, `build-job.yaml`, `cluster-alias.sh` and `dockerconfig.json.tpl`, and `grep -cE '^[[:space:]]*hack/build-cluster\.sh' Makefile.docker` returns ≥ 1 on an invocation line rather than a comment. Evidence: the five names and the count.
- [ ] **`upload` and `clean` become no-ops for a remote target** — `make -n buca BUILD_TARGET=nukeprod REF=<sha> VERSION=probe` prints neither `docker push` nor `docker rmi`, while `make -n buca` still prints both. Evidence: both outputs, plus the exit code of `make buca BUILD_TARGET=nukeprod REF=<sha> VERSION=<scratch-tag>`, which exits 0 at the end of an otherwise successful build rather than failing with `An image does not exist locally with the tag`.
- [ ] **The documented build path matches reality** — `README.md` carries a `## Build` section showing the exact cluster invocation `make build BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>`, states that the local arm is the default, that `buca` is a deploy command, and that `hack/build-cluster-bootstrap.sh` must be run once against a cluster before its first build; `CHANGELOG.md` carries a `## Unreleased` section with one bullet for this change. Evidence: `grep -c 'BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>' README.md` ≥ 1, `grep -c '^## Build$' README.md` = 1, `grep -c 'build-cluster-bootstrap.sh' README.md` ≥ 1, and `grep -c '^## Unreleased$' CHANGELOG.md` = 1.

## Verification

### Container-executable (runs inside the YOLO container at prompt time)

- `make precommit` — format, lint, vet and tests clean
- `make test` — unit suite passes
- `bash -n hack/build-cluster.sh && bash -n hack/cluster-alias.sh && bash -n hack/build-cluster-bootstrap.sh` — the copied runners parse
- `grep -c '^BUILD_TARGET' Makefile.docker` returns ≥ 1
- `grep -cE '^[[:space:]]*hack/build-cluster\.sh' Makefile.docker` returns ≥ 1
- `git ls-tree --name-only HEAD hack/` lists the five expected files
- `make -n build` prints `docker buildx build`, `--output type=docker,rewrite-timestamp=true,unpack=false`, `--provenance=false` and `--sbom=false`
- `make -n build BUILD_TARGET=nukeprod REF=<sha> VERSION=probe` prints `hack/build-cluster.sh`, `IMAGE='docker.io/bborbe/obsidian-redirector'` and `TAG='probe'`
- `make -n buca BUILD_TARGET=nukeprod REF=<sha> VERSION=probe` prints neither `docker push` nor `docker rmi`
- `grep -c '^## Build$' README.md` returns 1; `grep -c 'BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>' README.md` returns ≥ 1; `grep -c 'build-cluster-bootstrap.sh' README.md` returns ≥ 1
- `grep -c '^## Unreleased$' CHANGELOG.md` returns 1
- `grep -c '^override REPO_URL' Makefile.docker` returns 1 and `grep -c 'BUILD_REPO_URL ?=' Makefile.docker` returns 0
- `make -n build BUILD_TARGET=nukeprod REF=<sha> VERSION=probe REPO_URL=https://github.com/attacker/x.git` still prints `REPO_URL='https://github.com/bborbe/obsidian-redirector.git'` — the caller cannot repoint the Job
- `grep -c 'automountServiceAccountToken: false' hack/build-job.yaml` returns 1
- `grep -c '^SOURCE_DATE_EPOCH' Makefile.docker` returns ≥ 1

### Operator-executable (runs on the host after the change lands on master, spec verification ladder)

- `make build BUILD_TARGET=nukeprod REF=<sha> VERSION=<scratch-tag>` — one real cluster build, Job name, node and digest recorded
- `make build BUILD_TARGET=nukeprod REF=<sha>` with no `VERSION` — exits non-zero naming `VERSION`, and `kubectlnukeprod -n build get jobs` shows no new Job
- `crane digest docker.io/bborbe/obsidian-redirector:<scratch-tag>` — the push landed
- `docker image inspect docker.io/bborbe/obsidian-redirector:<scratch-tag>` — must fail with `No such image` on this Mac
- `DOCKER_CONTEXT=orbstack make build` twice with the tag removed between runs — both `docker image inspect --format '{{.Id}}'` readings agree

## Desired Behavior

1. **A build target selects the arm.** `BUILD_TARGET` defaults to `local`. With `local`, `make build` builds on this machine and tags `$(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH)` — the same reference it tags today.
2. **Any other value routes to a cluster Job.** With `BUILD_TARGET` set to a cluster alias or kubectl context, `make build` renders the Job template and applies it through the vendored runner, waits for the Job, and reports success or failure with the Job's own log lines.
3. **The cluster arm needs a caller-supplied tag.** It refuses to run unless `VERSION` came from the caller — the command line or the environment. It refuses in particular when `VERSION` is undefined, so a tag nobody chose can never be pushed. The refusal names the variable and happens before any Job is created.
4. **Both arms publish to one repository.** The cluster arm uses `$(DOCKER_REGISTRY)/$(SERVICE):$(VERSION)` — the same repository the local arm pushes, differing only in tag.
5. **`upload` and `clean` stay local-only.** For a cluster `BUILD_TARGET` they expand to nothing, so `make buca BUILD_TARGET=<cluster>` ends without a push and without a local `rmi`, rather than failing on a push of an image that was never loaded locally.
6. **The cluster build works from a clone.** The Job clones the repository at `REF` and builds with module resolution, so no `vendor/` directory is required.
7. **The local arm is reproducible.** The local build goes through `docker buildx build` with `rewrite-timestamp` clamped to the commit epoch, `unpack=false`, and provenance/SBOM attestations disabled — the flags the copied arm ships with.
8. **The documented build path matches reality.** The README's build section shows both arms and the CHANGELOG records the change.

## Constraints

- The local arm's tag must not change: `$(DOCKER_REGISTRY)/$(SERVICE):$(BRANCH)`.
- The cluster arm passes the runner's required `IMAGE` input explicitly at the call site as `IMAGE='$(DOCKER_REGISTRY)/$(SERVICE)'`. `example.env:4` also exports `IMAGE=bborbe/obsidian-redirector`, which is unqualified — a dispatch that let that value reach the runner would push to `bborbe/obsidian-redirector` with no registry. The explicit assignment is what overrides it.
- `VERSION` has no definition anywhere in this repo, so the refusal must fire when `$(origin VERSION)` is `file` **or** `undefined`. A guard testing only for `file` never fires here.
- No `check-version-tag` prerequisite on `build:`. This repo has no such target, and adding one breaks the scratch-tag build.
- `SOURCE_DATE_EPOCH ?= $(shell git log -1 --format=%ct)` must be defined and passed as `--build-arg SOURCE_DATE_EPOCH=$(SOURCE_DATE_EPOCH)` on the local arm, alongside `BUILD_DATE ?= $(shell git log -1 --format=%cI)`. `rewrite-timestamp` clamps layer timestamps and the image's `.Created` to it; without it the flag clamps to nothing and the reproducibility AC cannot pass. **`Makefile.docker:13` must change too**: today it is `--build-arg BUILD_DATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ)`, an inline wall clock that overrides any variable — adding the variable and leaving that line keeps a per-run label and fails the reproducibility AC. Replace it with `--build-arg BUILD_DATE=$(BUILD_DATE)`, mirroring `go-skeleton@eb050aa2:Makefile.docker:43`.
- `REPO_URL` and `PUSH_SECRET` are assigned in the dispatch with `override`, not `:=` and not the copied `BUILD_REPO_URL ?=` / `BUILD_PUSH_SECRET ?=`. Both a plain `:=` and a `?=` are beaten by a command-line assignment, so either would let `make build BUILD_TARGET=nukeprod REPO_URL=<any repo>` render a push-credential-bearing Job that clones an arbitrary repository. `override` is the only form that holds.
- The five `hack/` files are copied unmodified, each pinned to a full commit SHA so the content cannot drift between spec approval and execution: `hack/build-cluster.sh`, `hack/cluster-alias.sh` and `hack/build-job.yaml` from `agent-task-executor` at `d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40` (its Job spec carries `automountServiceAccountToken: false`, which `go-skeleton`'s does not); `hack/build-cluster-bootstrap.sh` and `hack/dockerconfig.json.tpl` from `go-skeleton` at `eb050aa20bb6194d8af7e47baf8880df789bd1ca`. `dockerconfig.json.tpl` is not optional — `build-cluster-bootstrap.sh:11` resolves it to build the dockerconfigjson secret the push authenticates with. The dark-factory container mounts only this repo, so each file is fetched from `https://raw.githubusercontent.com/bborbe/<repo>/<sha>/hack/<file>`.
- The Job template passes `DOCKER_REGISTRY=docker.io` as a build argument so the Dockerfile's base images resolve from the public registry inside the cluster. Do not change that to the private mirror.
- `.gitignore` already ignores only `/specs/log` and not `/specs/`, so `specs/` needs no change. Leave the rest of `.gitignore` untouched.
- `README.md` is the in-repo home for the two-arm topology; the spec dies after implementation, the README does not.
- No new scenario. The behaviour is reachable by checks on the rendered invocation plus one real cluster build.

## Assumptions

These must hold before the cluster AC can pass; none is created by this spec.

- The `build` namespace and the `buildkit-push` dockerconfigjson secret exist in the target cluster, **and that credential covers Docker Hub** — it authenticates against `docker.io`, which is the registry this repo's `DOCKER_REGISTRY` names, so no mirror push is attempted by the arm.
- The repo is public, so the Job's anonymous `git fetch` succeeds; the init container carries no clone credential.
- `crane` is installed on the host running the ACs.
- `REF` exists in the local repository **and on `origin`**: the runner derives `SOURCE_DATE_EPOCH` from the local repository, and the Job's init container fetches the ref from the remote. A commit that exists only locally fails at the clone.
- The build pod can reach the Go module proxy.
- `kubectl` is configured for the target cluster through the alias files `hack/cluster-alias.sh` resolves. While sun is host-down the build runs on `nukeprod`; each evidence row records which cluster it used.
- The active `docker buildx` builder on this Mac is the **`docker` driver** (`docker buildx ls`, the `*` row). A `docker-container` builder is a separate build cache and the local-arm recipe was not measured against it; on such a builder the local AC fails at the buildx step rather than producing a wrong image.
- `hack/build-cluster-bootstrap.sh` has been run once against the target cluster. It is what creates the `build` namespace and the `buildkit-push` secret, and it is idempotent; this spec copies it and documents it but does not run it.

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---|---|---|
| `BUILD_TARGET` names an unknown cluster or context | The runner fails before applying anything and lists the accepted values | Re-run with a listed target; no Job exists to clean up |
| `VERSION` omitted on the cluster arm | Refusal before the Job is created, naming `VERSION` | Re-run passing `VERSION=<scratch-tag>` |
| `REF` is a short SHA, or is not in the local repository | The runner dies resolving `SOURCE_DATE_EPOCH` | `git fetch`, then re-run with the full 40-character SHA |
| `REF` exists locally but was never pushed | The clone init container fails with exit 128: the ref does not exist on `origin` | Push the branch or commit, then re-run |
| The clone's commit epoch differs from the locally derived one | The build container refuses before `buildctl` runs and names both epochs | `git fetch`, then re-run |
| A Job with the derived name already exists in `build` | The runner refuses and names the existing Job rather than overwriting it | Delete the finished Job, then re-run |
| The push fails `401 Unauthorized` after a successful build | The mounted secret is for a different registry | Check the secret's `auths` keys with `kubectlnukeprod -n build get secret buildkit-push -o jsonpath='{.data.\.dockerconfigjson}' \| base64 -d \| jq -r '.auths \| keys[]'` — keys only, never the values — and confirm the key has no port suffix that the reference lacks |
| The build exceeds `BUILD_TIMEOUT` | The runner exits `timeout` after `TIMEOUT` + 120 s, and the Job keeps running in the cluster | Inspect `kubectlnukeprod -n build get jobs`, read the pod log, delete the Job once finished, then re-run with a larger `BUILD_TIMEOUT` |
| Docker Hub rate-limits the base-image pulls or the push | The Job fails with `toomanyrequests` / `429` in its log; the runner prints the last 100 log lines and the Job name | Wait for the window to reset and re-run; the local arm is unaffected because it pulls through the Mac's own cache |
| The local buildx builder is not the `docker` driver | `make build` fails at the buildx step before any image is produced | `docker buildx ls`, switch to the `docker`-driver builder (the `*` row), then re-run |
| The runner is interrupted after `apply` | The Job keeps running in the cluster while the local invocation exits; the next run then hits the existing-Job refusal | Inspect `kubectlnukeprod -n build get jobs`, read the pod's log, delete the Job once finished |
| The build pod cannot reach the module proxy | The Job fails; the runner prints the last 100 log lines and the Job name | Inspect the Job logs; the local arm is unaffected |
| The build pod is OOM-killed or throttled | The Job reports `failed`; the pod status carries `OOMKilled` or a CPU-throttling event | Raise the template's limits in the same change, or build locally for that one image |
| The local arm regresses | `make build` fails, or tags a different reference | Revert the arm; the local path is the regression gate |

## Security / Abuse Cases

- The runner substitutes caller-supplied values into a manifest and into embedded shell, so every value is checked against a character whitelist and rejected for embedded newlines before rendering. Do not add a value path that bypasses that check.
- The repository URL and the push-secret name are pinned with `override`, not a plain `:=`. A plain `:=` is **overridden by a command-line assignment** in GNU make — measured on this branch: with `REPO_URL :=`, `make build REPO_URL=https://github.com/attacker/x.git` renders that URL into the Job. `override REPO_URL :=` is the only assignment form a caller cannot beat, and the same applies to `PUSH_SECRET`. A configurable clone source on a build that mounts a push credential is the hazard this constraint removes.
- The build Job mounts a push credential from a `build`-namespace secret and runs with `automountServiceAccountToken: false`. The credential is scoped to pushing an image; it grants nothing else.
- The build container runs as an unprivileged user with the sandbox profiles relaxed because rootless BuildKit requires it. That relaxation is deliberate and must not be copied to the clone init container.
- `VERSION` is caller-controlled and becomes an image tag and a Job name. The tag is whitelisted and the Job name is sanitized to a valid DNS name, truncated, and stripped of trailing separators.

## Suggested Decomposition

Prompts should be generated in this order — each row is a single prompt with a clear scope.

| # | Prompt focus | Covers DBs | Covers ACs | Depends on |
|---|---|---|---|---|
| 1 | `Makefile.docker` dispatch: the `BUILD_TARGET` split, the `VERSION` refusal, the composed reference, local-only `upload`/`clean`, and the buildx local branch | 1, 2, 3, 4, 5, 7 | 1, 2, 4, 5, 7 | — |
| 2 | The build path itself: the five `hack/` files | 2, 6 | 3, 6 | prompt 1 (the dispatch calls the runner) |
| 3 | Docs: the README build section and the CHANGELOG bullet | 8 | 8 | — |

Rationale: prompt 1 establishes the dispatch and the refusal that makes any of it reachable; prompt 2 supplies the runner the dispatch calls; prompt 3 is independent and touches no build logic.

## Do-Nothing Option

Leaving this repo without a cluster arm keeps its image tied to the Mac. Every rebuild, every release cut while the operator is away from this machine, and every future unattended pipeline needs a human at the keyboard for this one repo — while its siblings build themselves. The cost is not the one build; it is that the automation has a hole shaped exactly like the repo nobody is watching.
