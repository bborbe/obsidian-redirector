// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"os"
	"time"

	libhttp "github.com/bborbe/http"
	"github.com/bborbe/log"
	libmetrics "github.com/bborbe/metrics"
	"github.com/bborbe/run"
	libsentry "github.com/bborbe/sentry"
	"github.com/bborbe/service"
	libtime "github.com/bborbe/time"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/bborbe/obsidian-redirector/pkg/factory"
	"github.com/bborbe/obsidian-redirector/pkg/handler"
)

func main() {
	app := &application{}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}

type application struct {
	SentryDSN       string            `required:"true"  arg:"sentry-dsn"        env:"SENTRY_DSN"        usage:"SentryDSN"                                                                   display:"length"`
	SentryProxy     string            `required:"false" arg:"sentry-proxy"      env:"SENTRY_PROXY"      usage:"Sentry Proxy"`
	Listen          string            `required:"false" arg:"listen"            env:"LISTEN"            usage:"admin listen address"                                                                         default:":9090"`
	PublicListen    string            `required:"false" arg:"public-listen"     env:"PUBLIC_LISTEN"     usage:"public listen address"                                                                        default:":8080"`
	VaultAllowlist  string            `required:"true"  arg:"vault-allowlist"   env:"VAULT_ALLOWLIST"   usage:"Comma separated list of vault names the redirector may emit targets for"`
	FileAllowlist   string            `required:"true"  arg:"file-allowlist"    env:"FILE_ALLOWLIST"    usage:"Comma separated list of vault-relative path prefixes a file must fall under"`
	BuildGitVersion string            `required:"false" arg:"build-git-version" env:"BUILD_GIT_VERSION" usage:"Build Git version"                                                                            default:"dev"`
	BuildGitCommit  string            `required:"false" arg:"build-git-commit"  env:"BUILD_GIT_COMMIT"  usage:"Build Git commit hash"                                                                        default:"none"`
	BuildDate       *libtime.DateTime `required:"false" arg:"build-date"        env:"BUILD_DATE"        usage:"Build timestamp (RFC3339)"`
}

func (a *application) Run(ctx context.Context, sentryClient libsentry.Client) error {
	libmetrics.NewBuildInfoMetrics().SetBuildInfo(a.BuildGitVersion, a.BuildGitCommit, a.BuildDate)

	return service.Run(
		ctx,
		a.createAdminHTTPServer(sentryClient),
		a.createPublicHTTPServer(),
	)
}

// createAdminHTTPServer serves the canonical admin endpoints on the admin
// listener (port 9090 by default). It carries no business routes: the gateway
// mounts this server under `/admin/<service>/...`, so a business endpoint placed
// here would be reachable only through the admin prefix rather than at the root
// the public hostname serves.
func (a *application) createAdminHTTPServer(sentryClient libsentry.Client) run.Func {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		router := mux.NewRouter()
		router.Path("/healthz").Handler(factory.CreateHealthzHandler())
		router.Path("/readiness").Handler(libhttp.NewPrintHandler("OK"))
		router.Path("/metrics").Handler(promhttp.Handler())
		router.Path("/setloglevel/{level}").
			Handler(log.NewSetLoglevelHandler(ctx, log.NewLogLevelSetter(2, 5*time.Minute)))
		router.Path("/gc").Handler(libhttp.NewGarbageCollectorHandler())
		router.Path("/testloglevel").Handler(factory.CreateTestLoglevelHandler())
		router.Path("/sentryalert").Handler(factory.CreateSentryAlertHandler(sentryClient))

		glog.V(2).Infof("starting admin http server listen on %s", a.Listen)
		return libhttp.NewServer(a.Listen, router).Run(ctx)
	}
}

// createPublicHTTPServer serves the one public route, `/obsidian`, on its own
// listener. It is deliberately separate from the admin server: this endpoint is
// unauthenticated and reachable from the public internet, whereas the admin
// endpoints sit behind the gateway's auth. Sharing a router would put an
// unauthenticated redirect next to destructive admin handlers.
func (a *application) createPublicHTTPServer() run.Func {
	return func(ctx context.Context) error {
		vaults := handler.VaultAllowlist(handler.ParseAllowlist(a.VaultAllowlist))
		files := handler.FileAllowlist(handler.ParseFileAllowlist(a.FileAllowlist))

		// An empty allowlist allows nothing — the handler fails closed, so a
		// missing configuration rejects every request rather than permitting
		// every vault.
		if len(vaults) == 0 || len(files) == 0 {
			glog.Warningf(
				"allowlist is empty (vaults=%d files=%d): every request will be rejected",
				len(vaults),
				len(files),
			)
		}

		router := mux.NewRouter()
		router.Path("/obsidian").
			Handler(factory.CreateObsidianRedirectHandler(vaults, files))

		glog.V(2).Infof("starting public http server listen on %s", a.PublicListen)
		return libhttp.NewServer(a.PublicListen, router).Run(ctx)
	}
}
