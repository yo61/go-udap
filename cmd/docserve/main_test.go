package main

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testSite() fstest.MapFS {
	return fstest.MapFS{
		"index.html":                    {Data: []byte("home")},
		"404.html":                      {Data: []byte("custom not found")},
		"concepts/index.html":           {Data: []byte("concepts")},
		"_next/static/app.js":           {Data: []byte("js")},
		"api/search":                    {Data: []byte(`{"ok":true}`)},
		"empty/.keep":                   {Data: nil},
		"concepts/multi-nic/index.html": {Data: []byte("multi-nic")},
	}
}

func get(t *testing.T, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestHandlerServesPagesUnderBasePath(t *testing.T) {
	h := newHandler(testSite())
	cases := map[string]string{
		"/go-udap/":                    "home",
		"/go-udap/concepts/":           "concepts",
		"/go-udap/concepts/multi-nic/": "multi-nic",
		"/go-udap/_next/static/app.js": "js",
		"/go-udap/api/search":          `{"ok":true}`,
	}
	for target, want := range cases {
		rec := get(t, h, target)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: status %d, want 200", target, rec.Code)
			continue
		}
		if got := rec.Body.String(); got != want {
			t.Errorf("GET %s: body %q, want %q", target, got, want)
		}
	}
}

func TestHandlerRedirects(t *testing.T) {
	h := newHandler(testSite())
	cases := []struct {
		target   string
		status   int
		location string
	}{
		{"/", http.StatusFound, "/go-udap/"},
		{"/go-udap", http.StatusMovedPermanently, "/go-udap/"},
		{"/go-udap/concepts", http.StatusMovedPermanently, "concepts/"},
	}
	for _, c := range cases {
		rec := get(t, h, c.target)
		if rec.Code != c.status {
			t.Errorf("GET %s: status %d, want %d", c.target, rec.Code, c.status)
		}
		if got := rec.Header().Get("Location"); got != c.location {
			t.Errorf("GET %s: Location %q, want %q", c.target, got, c.location)
		}
	}
}

func TestHandlerServesCustom404(t *testing.T) {
	h := newHandler(testSite())
	for _, target := range []string{
		"/go-udap/nope",
		"/go-udap/concepts/nope/",
		"/go-udap/empty/",
		"/elsewhere",
	} {
		rec := get(t, h, target)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: status %d, want 404", target, rec.Code)
		}
		if got := rec.Body.String(); got != "custom not found" {
			t.Errorf("GET %s: body %q, want the site's 404.html", target, got)
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("GET %s: Content-Type %q, want text/html", target, ct)
		}
	}
}

func TestHandlerFallsBackWhenSiteHasNo404Page(t *testing.T) {
	site := testSite()
	delete(site, "404.html")
	rec := get(t, newHandler(site), "/go-udap/nope")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", rec.Code)
	}
}

type brokenFS struct{}

func (brokenFS) Open(string) (fs.File, error) { return nil, errors.New("disk on fire") }

func TestHandlerReportsUnexpectedFSErrors(t *testing.T) {
	rec := get(t, newHandler(brokenFS{}), "/go-udap/")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status %d, want 500", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "disk on fire") {
		t.Errorf("body %q does not name the underlying error", body)
	}
}

func TestCheckSiteRejectsMissingBuild(t *testing.T) {
	err := checkSite(fstest.MapFS{}, "docs/site/out")
	if err == nil {
		t.Fatal("checkSite accepted a directory with no index.html")
	}
	if !strings.Contains(err.Error(), "pnpm build") {
		t.Errorf("error %q does not say how to fix it", err)
	}
}

func TestCheckSiteRejectsBuildWithoutBasePath(t *testing.T) {
	site := fstest.MapFS{"index.html": {Data: []byte(`<script src="/_next/static/a.js">`)}}
	err := checkSite(site, "docs/site/out")
	if err == nil {
		t.Fatal("checkSite accepted a build whose assets are not under /go-udap")
	}
	if !strings.Contains(err.Error(), "BASE_PATH") {
		t.Errorf("error %q does not point at BASE_PATH", err)
	}
}

func TestCheckSiteAcceptsBasePathBuild(t *testing.T) {
	site := fstest.MapFS{"index.html": {Data: []byte(`<script src="/go-udap/_next/static/a.js">`)}}
	if err := checkSite(site, "docs/site/out"); err != nil {
		t.Errorf("checkSite: %v", err)
	}
}
