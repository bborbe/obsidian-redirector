// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/bborbe/errors"
	"github.com/bborbe/validation"
	"github.com/golang/glog"
)

// obsidianScheme is the only URL scheme this service may emit. It is a constant
// rather than a variable so no code path can widen it at runtime.
const obsidianScheme = "obsidian://open"

// VaultAllowlist is the set of Obsidian vault names this redirector will emit
// targets for. Without it a caller could name any vault, and the emitted
// deeplink would be built from an unvalidated value.
type VaultAllowlist []string

// Contains reports whether name is a member of the allowlist.
func (v VaultAllowlist) Contains(name string) bool {
	for _, item := range v {
		if item == name {
			return true
		}
	}
	return false
}

// FileAllowlist is the set of vault-relative path prefixes a `file` value must
// fall under. It is the second half of the allowlist pair: the vault allowlist
// constrains which vault, this constrains which paths inside it.
type FileAllowlist []string

// Contains reports whether file falls under any allowed prefix.
func (f FileAllowlist) Contains(file string) bool {
	for _, prefix := range f {
		if strings.HasPrefix(file, prefix) {
			return true
		}
	}
	return false
}

// ParseAllowlist splits a comma-separated configuration value into its entries,
// trimming surrounding whitespace and dropping empty ones. An empty input
// yields an empty list, which allows nothing — a missing allowlist fails closed
// rather than open.
func ParseAllowlist(value string) []string {
	result := make([]string, 0, len(value))
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// filePrefixSeparator is the separator vault-relative file entries use. It is
// hard-coded rather than taken from os.PathSeparator because these are Obsidian
// vault paths, which are always slash-separated regardless of the host OS.
const filePrefixSeparator = "/"

// ParseFileAllowlist splits a comma-separated configuration value into the path
// prefixes a `file` value may fall under, normalising every entry to end in a
// separator.
//
// It is a separate seam from ParseAllowlist on purpose. FileAllowlist.Contains
// is a prefix match, so an entry without a trailing separator widens the match
// into any sibling whose name merely starts with the same characters — `tasks`
// would admit `tasksExtra/`. Normalising inside ParseAllowlist instead would
// stamp the separator onto vault names, and VaultAllowlist.Contains is exact
// membership, so every request would be rejected.
func ParseFileAllowlist(value string) FileAllowlist {
	entries := ParseAllowlist(value)
	result := make(FileAllowlist, 0, len(entries))
	for _, entry := range entries {
		result = append(result, ensureFilePrefix(entry))
	}
	return result
}

// ensureFilePrefix returns entry with a trailing separator, so a prefix match
// cannot spill into a sibling directory whose name merely shares its opening
// characters. "25 Tasks" and "25 Tasks/" both yield "25 Tasks/".
func ensureFilePrefix(entry string) string {
	if strings.HasSuffix(entry, filePrefixSeparator) {
		return entry
	}
	return entry + filePrefixSeparator
}

// NewObsidianRedirectHandler creates an HTTP handler that answers
// `GET /obsidian?vault=<v>&file=<p>` with a 302 whose Location is the
// equivalent `obsidian://open` deeplink.
//
// Telegram rejects a `text_link` entity whose URL is not http/https
// ("Unsupported URL protocol"), so a deeplink cannot be attached to a message
// directly. This handler is the other half of that workaround: it accepts a URL
// Telegram will carry and converts it, on the tap, into one Obsidian opens.
//
// The handler only ever emits an `obsidian://open` target. There is
// deliberately no `?url=` passthrough — a parameter-derived redirect target
// would turn a public hostname into an open redirect usable as a phishing hop,
// and it would also lose `file=` to the outer query parser. A `url` parameter
// is rejected outright rather than ignored, so the guard fails loudly if one is
// ever reintroduced.
func NewObsidianRedirectHandler(
	vaults VaultAllowlist,
	files FileAllowlist,
) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if req.URL.Query().Has("url") {
			glog.V(2).Infof("reject obsidian redirect: url parameter is not supported")
			writeRejection(resp, "url parameter is not supported")
			return
		}

		vault := req.URL.Query().Get("vault")
		file := req.URL.Query().Get("file")

		if err := validateRedirectRequest(ctx, vaults, files, vault, file); err != nil {
			glog.V(2).Infof("reject obsidian redirect vault(%s) file(%s): %v", vault, file, err)
			writeRejection(resp, "invalid request")
			return
		}

		target := BuildObsidianURL(vault, file)
		glog.V(3).Infof("redirect obsidian vault(%s) file(%s)", vault, file)

		// Set the Location directly rather than going through http.Redirect:
		// http.Redirect parses and re-serialises the target, and the criterion
		// here is that the emitted Location is byte-for-byte the encoded
		// deeplink. Writing it verbatim removes that re-encoding step entirely.
		resp.Header().Set("Location", target)
		resp.WriteHeader(http.StatusFound)
	})
}

// writeRejection writes a 400 and logs if the write itself fails. A rejection
// must never be silently swallowed: a client that receives 200 from a rejected
// request would treat a broken link as a delivered one.
func writeRejection(resp http.ResponseWriter, message string) {
	resp.WriteHeader(http.StatusBadRequest)
	if _, err := resp.Write([]byte(message)); err != nil {
		glog.V(2).Infof("write rejection response failed: %v", err)
	}
}

// BuildObsidianURL returns the `obsidian://open` deeplink for a vault and a
// vault-relative file path, with both values percent-encoded.
func BuildObsidianURL(vault string, file string) string {
	return obsidianScheme + "?vault=" + EncodeQueryValue(vault) + "&file=" + EncodeQueryValue(file)
}

// EncodeQueryValue percent-encodes a value for the deeplink's query string.
//
// `url.QueryEscape` is used because it escapes the full set correctly for a
// query — `/` as `%2F`, `&` as `%26`, `?` as `%3F`, `#` as `%23` — and encodes a
// literal `+` as `%2B`. The one thing it gets wrong for this consumer is the
// space: it emits `+`, and **Obsidian does not decode `+` in a query string**.
// A link built with `+` opens nothing, and nothing in the URL looks wrong,
// which is why the bug survives review. Swapping `+` for `%20` is safe
// precisely because a literal `+` was already escaped to `%2B` above, so the
// replacement cannot corrupt one.
func EncodeQueryValue(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

// validateRedirectRequest applies the allowlist pair plus the structural checks
// that keep an emitted path inside the vault.
func validateRedirectRequest(
	ctx context.Context,
	vaults VaultAllowlist,
	files FileAllowlist,
	vault string,
	file string,
) error {
	return validation.All{
		validation.Name("vault", validation.NotEmptyString(vault)),
		validation.Name("file", validation.NotEmptyString(file)),
		validation.Name(
			"vault allowlist",
			validation.HasValidationFunc(func(ctx context.Context) error {
				if !vaults.Contains(vault) {
					return errors.Errorf(ctx, "vault %q is not in the allowlist", vault)
				}
				return nil
			}),
		),
		validation.Name(
			"file allowlist",
			validation.HasValidationFunc(func(ctx context.Context) error {
				if !files.Contains(file) {
					return errors.Errorf(ctx, "file %q is not under an allowed prefix", file)
				}
				return nil
			}),
		),
		validation.Name(
			"file relative",
			validation.HasValidationFunc(func(ctx context.Context) error {
				if strings.HasPrefix(file, "/") {
					return errors.Errorf(ctx, "file %q must be vault-relative", file)
				}
				for _, segment := range strings.Split(file, "/") {
					if segment == ".." {
						return errors.Errorf(ctx, "file %q must not traverse upward", file)
					}
				}
				return nil
			}),
		),
		// Control characters are rejected on both values. A `file` of
		// "25 Tasks/My Note\r\nLocation: https://evil.example" passes every
		// allowlist and traversal check above — it does start with an allowed
		// prefix and contains no `..` — and reaches the Location header. This
		// was found by the open-redirect falsifier, which caught the handler
		// returning 302 to that input rather than a rejection.
		validation.Name(
			"vault control characters",
			validation.HasValidationFunc(func(ctx context.Context) error {
				return rejectControlCharacters(ctx, "vault", vault)
			}),
		),
		validation.Name(
			"file control characters",
			validation.HasValidationFunc(func(ctx context.Context) error {
				return rejectControlCharacters(ctx, "file", file)
			}),
		),
	}.Validate(ctx)
}

// rejectControlCharacters returns an error if value contains any control
// character, including CR and LF. Header injection needs a newline, so refusing
// them at the boundary means the Location header can never be split.
func rejectControlCharacters(ctx context.Context, field string, value string) error {
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return errors.Errorf(ctx, "%s %q must not contain control characters", field, value)
		}
	}
	return nil
}
