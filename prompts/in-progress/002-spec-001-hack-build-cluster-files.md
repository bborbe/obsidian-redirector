---
status: approved
spec: [001-cluster-build-arm]
created: "2026-11-09T17:55:00Z"
queued: "2026-11-09T17:06:01Z"
branch: dark-factory/cluster-build-arm
---

# Vendor the five cluster build runner files under hack/

<summary>
- The cluster build's helper scripts and Job template are now vendored in this repo.
- Each file is pinned to an exact upstream commit, so its content cannot drift.
- The Job template runs without an auto-mounted service-account token.
- The bootstrap script provisions the namespace and push secret once per cluster.
- The runner validates every caller-supplied value before rendering the Job.
- No build logic changes here; this prompt only supplies the files the dispatch calls.
</summary>

<objective>
Copy the five cluster build runner files into `hack/`, byte-for-byte from their pinned upstream commits, so the dispatch written in the previous prompt can render and run a build Job. The files are copied unmodified; do not edit their contents.
</objective>

<context>
Read these before starting:
- `Makefile.docker` — the cluster branch invokes `hack/build-cluster.sh` and sources `hack/cluster-alias.sh` from it.
- `docs/dod.md` — Definition of Done.

The files come from two sibling repositories at full commit SHAs (the container mounts only this repo, so fetch each from its raw URL):
- `bborbe/agent-task-executor` at `d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40`
- `bborbe/go-skeleton` at `eb050aa20bb6194d8af7e47baf8880df789bd1ca`

The `agent-task-executor` Job spec is used because it carries `automountServiceAccountToken: false`; `go-skeleton`'s does not.
</context>

<requirements>
1. Create the directory `hack/` if it does not exist.

2. Fetch each file with `curl -fsSL <url> -o <path>`, using these exact pinned URLs and destinations:
   - `https://raw.githubusercontent.com/bborbe/agent-task-executor/d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40/hack/build-cluster.sh` → `hack/build-cluster.sh`
   - `https://raw.githubusercontent.com/bborbe/agent-task-executor/d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40/hack/cluster-alias.sh` → `hack/cluster-alias.sh`
   - `https://raw.githubusercontent.com/bborbe/agent-task-executor/d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40/hack/build-job.yaml` → `hack/build-job.yaml`
   - `https://raw.githubusercontent.com/bborbe/go-skeleton/eb050aa20bb6194d8af7e47baf8880df789bd1ca/hack/build-cluster-bootstrap.sh` → `hack/build-cluster-bootstrap.sh`
   - `https://raw.githubusercontent.com/bborbe/go-skeleton/eb050aa20bb6194d8af7e47baf8880df789bd1ca/hack/dockerconfig.json.tpl` → `hack/dockerconfig.json.tpl`

   `hack/dockerconfig.json.tpl` is not optional: `hack/build-cluster-bootstrap.sh` resolves it (via `SCRIPT_DIR`) to build the dockerconfigjson secret the push authenticates with.

3. Do not edit any of the five files. They are copied unmodified; their content is pinned by the commit SHA in the URL.

4. Make the three shell scripts executable:
   `chmod +x hack/build-cluster.sh hack/cluster-alias.sh hack/build-cluster-bootstrap.sh`
   `hack/build-job.yaml` and `hack/dockerconfig.json.tpl` stay non-executable data files.

5. Verify the copies are byte-identical to the pinned upstream content:
   ```bash
   sha256sum -c <<'EOF'
   976d4fb2be48fd25db09aa5b72a994ac7ae92269e1f75d1ade60eceef1c9b6b5  hack/build-cluster.sh
   292922f77702353bbf9166a3c2fd0d406d0c94246a90b93852c3cf3bea118448  hack/cluster-alias.sh
   e2a7d9d28dc01154ae184d681ff5798c5d256597f500adf1017eaa15089f81a9  hack/build-job.yaml
   f2d74f6001005b103f759ad1e30dfc661625af67f4303bc898364b8ed0a1a957  hack/build-cluster-bootstrap.sh
   bcf89cfc8cba8b36a2f0b33fdc28a1096bad349dc5207e5e23d417939433bc7a  hack/dockerconfig.json.tpl
   EOF
   ```

6. Acceptance criteria covered by this prompt:
   - `hack/` contains exactly the five files named above.
   - `bash -n` parses each of the three shell scripts.
   - `hack/build-job.yaml` contains exactly one `automountServiceAccountToken: false`.
   - `hack/dockerconfig.json.tpl` exists.

   The post-commit check `git ls-tree --name-only HEAD hack/` lists the five names. That check only passes after dark-factory commits this change, so it is recorded on the spec's Acceptance Criteria rather than run against the uncommitted working tree here.
</requirements>

<constraints>
- The five `hack/` files are copied unmodified, each pinned to a full commit SHA so the content cannot drift between spec approval and execution.
- `hack/build-cluster.sh`, `hack/cluster-alias.sh` and `hack/build-job.yaml` come from `agent-task-executor` at `d3d6e9b779f55ba4b4f9a9f03bceab2fa46d1e40` (its Job spec carries `automountServiceAccountToken: false`).
- `hack/build-cluster-bootstrap.sh` and `hack/dockerconfig.json.tpl` come from `go-skeleton` at `eb050aa20bb6194d8af7e47baf8880df789bd1ca`.
- The Job template passes `DOCKER_REGISTRY=docker.io` as a build argument; do not change that to a private mirror.
- Do NOT modify `.gitignore`; it already ignores only `/specs/log` and not `/specs/`.
- No new scenario.
- Do NOT commit — dark-factory handles git.
- Existing tests must still pass; do not modify Go source.
</constraints>

<verification>
Run from the repo root. All commands are container-executable.

```bash
set -e
test -f hack/build-cluster.sh
test -f hack/cluster-alias.sh
test -f hack/build-job.yaml
test -f hack/build-cluster-bootstrap.sh
test -f hack/dockerconfig.json.tpl
bash -n hack/build-cluster.sh
bash -n hack/cluster-alias.sh
bash -n hack/build-cluster-bootstrap.sh
grep -c 'automountServiceAccountToken: false' hack/build-job.yaml
sha256sum -c <<'EOF'
976d4fb2be48fd25db09aa5b72a994ac7ae92269e1f75d1ade60eceef1c9b6b5  hack/build-cluster.sh
292922f77702353bbf9166a3c2fd0d406d0c94246a90b93852c3cf3bea118448  hack/cluster-alias.sh
e2a7d9d28dc01154ae184d681ff5798c5d256597f500adf1017eaa15089f81a9  hack/build-job.yaml
f2d74f6001005b103f759ad1e30dfc661625af67f4303bc898364b8ed0a1a957  hack/build-cluster-bootstrap.sh
bcf89cfc8cba8b36a2f0b33fdc28a1096bad349dc5207e5e23d417939433bc7a  hack/dockerconfig.json.tpl
EOF
make precommit
```

Before finishing, re-run the block above and confirm it passes; walk each acceptance criterion in requirement 6 against the change.
</verification>

<!-- OPEN QUESTION (auditor/reviewer): this prompt assumes raw.githubusercontent.com is reachable from the YOLO container for the two sibling repos. The spec's Constraints section explicitly specifies fetching from these raw URLs, so the assumption is inherited from the spec. If the container's proxy only permits the project's own remote, the files must be supplied another way (for example, pre-staged into the repo by the operator) and the sha256 checks in requirement 5 are the integrity guard. -->
<!-- OPEN QUESTION (auditor/reviewer): the spec's container-executable verification lists `git ls-tree --name-only HEAD hack/`, but HEAD cannot contain these files until dark-factory commits this prompt's change. This prompt uses `test -f` for the prompt-time check and leaves the `git ls-tree` assertion to the post-commit spec verification. -->
