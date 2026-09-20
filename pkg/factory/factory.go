// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package factory

import (
	"net/http"

	libsentry "github.com/bborbe/sentry"

	"github.com/bborbe/obsidian-redirector/pkg/handler"
)

// CreateTestLoglevelHandler creates an HTTP handler that tests different glog verbosity levels.
func CreateTestLoglevelHandler() http.Handler {
	return handler.NewTestLoglevelHandler()
}

// CreateSentryAlertHandler creates an HTTP handler that sends test alerts to Sentry.
func CreateSentryAlertHandler(sentryClient libsentry.Client) http.Handler {
	return handler.NewSentryAlertHandler(sentryClient)
}

// CreateHealthzHandler creates an HTTP handler that serves the canonical
// `/healthz` liveness response (HTTP 200, body `{"status":"ok"}`,
// Content-Type: application/json).
func CreateHealthzHandler() http.Handler {
	return handler.NewHealthzHandler()
}

// CreateObsidianRedirectHandler creates an HTTP handler that converts a
// Telegram-carriable `https://` request into an `obsidian://open` deeplink.
// `vaults` and `files` are the allowlist pair that constrains what the handler
// is willing to emit; an empty allowlist allows nothing.
func CreateObsidianRedirectHandler(
	vaults handler.VaultAllowlist,
	files handler.FileAllowlist,
) http.Handler {
	return handler.NewObsidianRedirectHandler(vaults, files)
}
