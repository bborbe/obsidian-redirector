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

## Build

`make build` builds the image locally and is the default. It tags
`docker.io/bborbe/obsidian-redirector:<branch>` (the checked-out branch with `/`
replaced by `-`). The local build runs through `docker buildx build` with
`rewrite-timestamp` clamped to the built commit, so two builds of one commit
produce the same image — **provided the active buildx builder is the `docker`
driver** (`docker buildx ls`, the `*` row). A `docker-container` builder is a
separate build cache; on it the same recipe silently produces a different image
per run.

To build inside a Kubernetes cluster instead — unattended, from a committed ref —
pass a cluster alias or kubectl context and a tag:

```bash
make build BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>
```

The cluster arm renders a Kubernetes Job, waits for it, and pushes
`docker.io/bborbe/obsidian-redirector:<tag>` from inside the cluster. `VERSION`
has no default: the build refuses unless you supply it on the command line or in
the environment. `REF` must be a full 40-character commit SHA that exists on
`origin`, because the Job clones the repository at that ref.

`buca` is the deploy command. For a cluster build use
`make buca BUILD_TARGET=<cluster> REF=<sha> VERSION=<tag>`; the local `upload`
and `clean` steps are skipped because the image is pushed from inside the
cluster. Note that `make buca` does **not** apply manifests in this repo — its
`apply` target iterates component subdirectories and there are none here; the
deployment is applied from `bborbe/nuke`.

Before the first cluster build against a cluster, run the bootstrap once (it is
idempotent; it creates the `build` namespace and the `buildkit-push` secret). It
needs `teamvault-cli` on `PATH` and a teamvault config (`TEAMVAULT`, default
`~/.config/teamvault-cli/config.json`):

```bash
hack/build-cluster-bootstrap.sh <cluster>
```

## Deploy

Deployed as a raw-manifest component of the [nuke](https://github.com/bborbe/nuke) repo, alongside `notification-telegram`:

```bash
make buca
```
