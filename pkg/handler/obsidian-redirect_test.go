// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler_test

import (
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/obsidian-redirector/pkg/handler"
)

var _ = Describe("ObsidianRedirectHandler", func() {
	var httpHandler http.Handler

	BeforeEach(func() {
		httpHandler = handler.NewObsidianRedirectHandler(
			handler.VaultAllowlist{"Personal", "Trading"},
			handler.FileAllowlist{"25 Tasks/", "24 Goals/"},
		)
	})

	// request issues a GET with the given raw (still-encoded) query string.
	request := func(rawQuery string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/obsidian?"+rawQuery, nil)
		resp := httptest.NewRecorder()
		httpHandler.ServeHTTP(resp, req)
		return resp
	}

	Context("a valid request", func() {
		It("returns 302", func() {
			resp := request("vault=Personal&file=25%20Tasks%2FMy%20Note")
			Expect(resp.Code).To(Equal(http.StatusFound))
		})

		It("emits a Location that is byte-for-byte the encoded deeplink", func() {
			resp := request("vault=Personal&file=25%20Tasks%2FMy%20Note")
			// Asserted on the RAW header. Parsing it through url.ParseQuery
			// first would decode `+` and `%20` to the same value, so a
			// round-trip assertion passes on the buggy form and proves nothing.
			Expect(resp.Header().Get("Location")).
				To(Equal("obsidian://open?vault=Personal&file=25%20Tasks%2FMy%20Note"))
		})

		It("encodes a space as %20 and never as +", func() {
			resp := request("vault=Personal&file=25%20Tasks%2FMy%20Note")
			location := resp.Header().Get("Location")
			Expect(location).To(ContainSubstring("%20"))
			// The whole point of the service: Obsidian does not decode `+`.
			Expect(location).NotTo(ContainSubstring("+"))
		})

		It("encodes the path separator as %2F", func() {
			resp := request("vault=Personal&file=25%20Tasks%2FMy%20Note")
			Expect(resp.Header().Get("Location")).To(ContainSubstring("25%20Tasks%2FMy%20Note"))
		})

		It("absorbs an incoming + and re-emits it as %20", func() {
			// A `+` in a query string means space, so an upstream that encoded
			// spaces as `+` is still parsed correctly and re-emitted as `%20`.
			resp := request("vault=Personal&file=25+Tasks%2FMy+Note")
			Expect(resp.Code).To(Equal(http.StatusFound))
			Expect(resp.Header().Get("Location")).
				To(Equal("obsidian://open?vault=Personal&file=25%20Tasks%2FMy%20Note"))
		})

		It("preserves a literal + as %2B rather than as a space", func() {
			resp := request("vault=Personal&file=25%20Tasks%2FA%2BB")
			Expect(resp.Code).To(Equal(http.StatusFound))
			Expect(resp.Header().Get("Location")).
				To(Equal("obsidian://open?vault=Personal&file=25%20Tasks%2FA%2BB"))
		})
	})

	Context("rejections", func() {
		It("rejects a vault outside the allowlist with 400", func() {
			resp := request("vault=NotMyVault&file=25%20Tasks%2FMy%20Note")
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Header().Get("Location")).To(BeEmpty())
		})

		It("rejects a file outside the path allowlist with 400", func() {
			resp := request("vault=Personal&file=99%20Elsewhere%2FSecret")
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Header().Get("Location")).To(BeEmpty())
		})

		It("rejects a file that traverses upward with 400", func() {
			resp := request("vault=Personal&file=25%20Tasks%2F..%2F..%2Fetc%2Fpasswd")
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Header().Get("Location")).To(BeEmpty())
		})

		It("rejects an absolute file with 400", func() {
			resp := request("vault=Personal&file=%2Fetc%2Fpasswd")
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Header().Get("Location")).To(BeEmpty())
		})

		It(
			"rejects a file containing CRLF, which would otherwise reach the Location header",
			func() {
				// Found by the open-redirect falsifier below, which caught the
				// handler answering 302 to this input rather than rejecting it: it
				// passes the allowlist (starts with an allowed prefix) and the
				// traversal check (no `..`), and the decoded value carries
				// `\r\nLocation: https://evil.example`.
				resp := request(
					"vault=Personal&file=25%20Tasks%2FMy%20Note%0d%0aLocation%3A%20https%3A%2F%2Fevil.example",
				)
				Expect(resp.Code).To(Equal(http.StatusBadRequest))
				Expect(resp.Header().Get("Location")).To(BeEmpty())
			},
		)

		It("rejects a vault containing CRLF", func() {
			resp := request("vault=Personal%0d%0aX%3A%20y&file=25%20Tasks%2FMy%20Note")
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("rejects a missing vault with 400", func() {
			resp := request("file=25%20Tasks%2FMy%20Note")
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("rejects a missing file with 400", func() {
			resp := request("vault=Personal")
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
		})

		It("rejects a url parameter outright with 400", func() {
			// Rejected rather than ignored, so the open-redirect guard fails
			// loudly if a passthrough is ever reintroduced.
			resp := request(
				"url=https%3A%2F%2Fevil.example&vault=Personal&file=25%20Tasks%2FMy%20Note",
			)
			Expect(resp.Code).To(Equal(http.StatusBadRequest))
			Expect(resp.Header().Get("Location")).To(BeEmpty())
		})
	})

	Context("open-redirect falsifier", func() {
		// Every hostile input a caller could try. The invariant under test is
		// that no input can produce a Location outside `obsidian://open` — if
		// any single case emits something else, the service is a phishing hop
		// on a public hostname and this spec fails.
		hostileQueries := []string{
			"url=https%3A%2F%2Fevil.example",
			"url=https%3A%2F%2Fevil.example&vault=Personal&file=25%20Tasks%2FMy%20Note",
			"vault=Personal&file=25%20Tasks%2FMy%20Note&url=%2F%2Fevil.example",
			"vault=https%3A%2F%2Fevil.example&file=25%20Tasks%2FMy%20Note",
			"vault=Personal&file=https%3A%2F%2Fevil.example",
			"vault=Personal&file=25%20Tasks%2FMy%20Note%0d%0aLocation%3A%20https%3A%2F%2Fevil.example",
			"vault=Personal&file=javascript%3Aalert(1)",
			"vault=Personal&file=%2F%2Fevil.example",
			"vault=&file=",
			"",
		}

		It("never emits a Location outside obsidian://open", func() {
			for _, query := range hostileQueries {
				location := request(query).Header().Get("Location")
				if location == "" {
					continue // a rejection carries no Location at all
				}
				Expect(location).To(HavePrefix("obsidian://open"),
					"query %q produced Location %q", query, location)
			}
		})

		It("returns a rejection status for every hostile input", func() {
			for _, query := range hostileQueries {
				Expect(request(query).Code).To(Equal(http.StatusBadRequest),
					"query %q was not rejected", query)
			}
		})
	})

	Context("EncodeQueryValue", func() {
		It("encodes a space as %20", func() {
			Expect(handler.EncodeQueryValue("a b")).To(Equal("a%20b"))
		})

		It("encodes a literal + as %2B", func() {
			Expect(handler.EncodeQueryValue("a+b")).To(Equal("a%2Bb"))
		})

		It("encodes a path separator as %2F", func() {
			Expect(handler.EncodeQueryValue("a/b")).To(Equal("a%2Fb"))
		})

		It("encodes & ? and # so they cannot escape the query", func() {
			Expect(handler.EncodeQueryValue("a&b")).To(Equal("a%26b"))
			Expect(handler.EncodeQueryValue("a?b")).To(Equal("a%3Fb"))
			Expect(handler.EncodeQueryValue("a#b")).To(Equal("a%23b"))
		})
	})

	Context("ParseAllowlist", func() {
		It("splits on commas and trims whitespace", func() {
			Expect(handler.ParseAllowlist(" Personal , Trading ")).
				To(Equal([]string{"Personal", "Trading"}))
		})

		It("drops empty entries", func() {
			Expect(handler.ParseAllowlist("Personal,,Trading,")).
				To(Equal([]string{"Personal", "Trading"}))
		})

		It("returns an empty list for empty input, which allows nothing", func() {
			Expect(handler.ParseAllowlist("")).To(BeEmpty())
		})
	})
})
