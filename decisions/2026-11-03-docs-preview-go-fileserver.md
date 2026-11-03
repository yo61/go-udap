## Decision: Preview the docs export with an in-repo Go file server under /go-udap

`task docs:serve` runs `docs:build` (the CI build, with the `/go-udap`
basePath) and serves `docs/site/out` with `cmd/docserve`, a stdlib
`net/http` server mounted at `/go-udap/`. The `serve` npm devDependency, the
`start` / `start:local` scripts and the three pnpm overrides that existed only
for serve's tree (compression, brace-expansion, fast-uri) are removed.

## Context

Dependabot alert #14 (GHSA-g3hv-75wg-m249, minimatch `< 11.0.1`) hit
minimatch@3.1.5, pinned exactly by serve-handler@6.1.7 under serve@14.2.6. It
was a false positive (the vulnerable code starts at 7.0.0; dismissed as
inaccurate), but serve's tree had already needed three overrides, and an
override to ^11 would have broken serve-handler's `require('minimatch')(...)`
call. serve-handler gets only patch-level maintenance.

Probing the built site showed serve also did not behave like GitHub Pages:
`/concepts` was served 200 without the trailing-slash redirect Pages issues,
and `/concepts/index.html` redirected to `/concepts/index`. With
`trailingSlash: true`, that difference changes how relative links resolve, so
the preview could hide link bugs.

## Alternatives considered

- **Keep serve** with overrides and dismissals. Rejected: recurring supply-chain
  work for a preview that does not match production.
- **http-server**: matched Pages on redirects and 404.html, 0 advisories, but
  last released 2022-05 and depends on http-proxy (last released 2020): the same
  neglect risk as serve.
- **sirv-cli**: 13 packages and maintained, but no trailing-slash redirect and
  no custom 404 page.
- **python3 -m http.server**: no 404.html, and Python is not a project
  dependency.
- **Go server at `/` with a no-basePath build** (the old docs:serve shape).
  Rejected in favour of serving the exact CI artifact under `/go-udap`.

## Reasoning

Go is already required, so the npm dependency disappears without adding a
new one. Mounting at `/go-udap` makes `docs:serve` preview the same bytes
`.github/workflows/docs.yaml` publishes. Verified against the live site:
`/go-udap/concepts` → 301 to `/go-udap/concepts/`, misses → 404 with the site's
404.html.

## Trade-offs accepted

- ~100 lines of Go plus tests to maintain in `cmd/docserve`.
- `basePath` is duplicated as a constant (must match `next.config.mjs`);
  `checkSite` fails fast if the export was built without it.
- `/go-udap/x/index.html` redirects to `/go-udap/x/` (http.FileServer
  behaviour) where Pages serves it with 200. Harmless for preview.
- `/` redirects with 302, not 301, so a cached redirect cannot break
  `docs:dev`, which serves `/` on the same port.
- The local build's `SITE_URL` is the production one, so OG/metadata URLs
  point at yo61.github.io. Same as what CI publishes.

## Supersedes: none
