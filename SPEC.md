# goftpd HTTP serving

This document is the contract for how goftpd answers HTTP requests.
Implement the server against this file.

## Reader

You implement or change the HTTP handler.
After you read this file, you can decide the status code and the body
for any request.

## Terms

Use only the approved term for each idea.

| Idea | Term | Do not use |
| --- | --- | --- |
| `Config.Dir` | served directory | work dir, public |
| URL path with no matching file or directory | miss | 404 fallback, rewrite |
| HTML table of a directory | listing | index, browse, file manager |
| `--spa` | SPA mode | host mode, www mode |
| `/__lewkit__/` | asset prefix | reserved prefix, internal path, magic path |
| `index.html` in a directory | directory index | homepage, entry |
| `/index.html` at the served directory | site index | root index |
| `/404.html` at the served directory | site 404 | custom 404, error page |
| Templ HTML from goftpd | built-in page | internal template, our 404 |
| `asset.Mount` | asset handler | embed server |
| `sakuracss.Load` | stylesheet tag | vendored stylesheet |
| `a-h/templ` generated Go | generated file | compiled template |
| quoted Unix-nanosecond modification time | strong ETag | weak ETag, content hash |
| `Range` byte slice | byte range | partial download |

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--addr` | `:8080` | Listen address |
| `--dir` | `./` | Served directory |
| `--spa` | off | SPA mode |

## Request path

An asset URL is answered by the asset handler before these rules.

Use `r.URL.Path`.
Do not use `r.RequestURI`.
The query string is not part of the file path.

Decode `%xx` escapes.
Reject a path that escapes the served directory.
Open files through an `fs.FS` for the served directory.

Honor `GET` and `HEAD`.
For other methods, return `405`.

## Asset prefix

The asset handler serves lewkit assets in front of the file handler.

An asset URL is exactly one path segment under `/__lewkit__/`.
Every such URL belongs to the asset handler, registered or not.
It never reads the served directory.
It never enters SPA mode.
It never produces a listing.

| URL | Registered by |
| --- | --- |
| `/__lewkit__/sakura.css` | `github.com/lewtec/lewkit/x/http/asset/sakuracss` |

GET and HEAD of a registered name serve that file.
The asset handler sets `Cache-Control: max-age=5` and `Content-Type`.
Do not apply the strong ETag rules from "Real file" to an asset URL.

GET or HEAD of an unregistered name in that one segment returns status 404
with the net/http not-found body.
Do not render the built-in 404 page for that URL.

Any other method on an asset URL returns status 405 and `Allow: GET, HEAD`.
Do not use the localized 405 body for that URL.

A path that is not an asset URL is a normal request.
That includes `/__lewkit__`, `/__lewkit__/`, and a nested path such as
`/__lewkit__/docs/readme.txt`.
A served-directory file whose URL is one segment under `/__lewkit__/`,
including `/__lewkit__/sakura.css` and `/__lewkit__/notes.txt`, is not reachable.
The asset handler answers that URL.

Do not embed stylesheets in this module.
Do not add a `/__goftpd__/` handler. A path under that name is a normal request.

## Trailing slash

If the path names a directory and has no final `/`, send `308`
to the same path with `/`.
Do not redirect a miss.
Do not redirect a file.

## Real file

If the path names a regular file, serve that file with status 200.
Use `http.ServeContent` so `HEAD` and byte ranges work.

Set these headers before `ServeContent`:

- `Cache-Control: no-cache`. Do not set `max-age`. A client may store the
  bytes. It must revalidate. There is no freshness lifetime.
- `ETag`: a strong ETag for this URL. The opaque tag is the file's
  modification time as Unix nanoseconds, inside quotes (`"<ns>"`).
  Do not read the file to build it. Do not hash the bytes.
  A new modification time misses. The same modification time matches,
  even when the bytes changed.

`ServeContent` then:

- Seeks to a byte range and answers 206 with `Content-Range` and
  `Accept-Ranges: bytes`. One range is a single slice, not multipart.
  A range that does not overlap a non-empty file is 416.
- Answers 304 with an empty body when `If-None-Match` matches the strong
  ETag. `HEAD` is also empty.
- Honors `If-Range` when the strong ETag matches. Otherwise it sends the
  whole file with 200.
- Answers 412 when `If-Match` does not match.

`ServeContent` also sets `Last-Modified` from that same modification time.
When `If-None-Match` is present it decides, not `If-Modified-Since`.

Do not send a weak ETag. `If-Range` ignores weak validators.

A real file always wins over a listing, a directory index, and a miss.

## Directory, SPA mode off

Render the built-in listing with status 200.

The listing includes every name from `ReadDir`, including names that
start with `.`.
Do not hide `.git` or `.DS_Store`.
Do not show `.` as a row.
Show `..` as a parent link when the path is not `/`.

Sort directories first.
Then sort by name.
The sort is case-insensitive.

Columns: name, size, modified time.
A directory name ends with `/`.
A directory size is empty.
Format size as a decimal byte count with a unit (`B`, `KB`, `MB`, `GB`).
Format time as `2006-01-02 15:04`.

Breadcrumbs walk from `/` to the current path.
The first crumb is the `CrumbRoot` message and always links to `/`.
Each crumb is a link.

The listing is server-rendered HTML from templ.
The page loads no JavaScript.

The listing loads the Sakura stylesheet with the stylesheet tag.
Sakura styles the listing. Do not add an overlay stylesheet.

`index.html` in the directory is a row.
Do not serve it in place of the listing.

## Directory, SPA mode on

SPA mode never renders a listing.

If the directory has `index.html`, serve that file with status 200.
If it does not, treat the request as a miss.

## Miss, SPA mode off

Return the built-in 404 page with status 404.
Do not read `/404.html`.
Do not read `/index.html`.

## Miss, SPA mode on

Look only at the served-directory root.
Do not walk parent directories.

1. If `/404.html` exists, serve it with status 404.
2. Else if `/index.html` exists, serve it with status 200.
3. Else return the built-in 404 page with status 404.

When you serve `/404.html` with status 404, set `Content-Type` from the
file and copy the body.
Do not use `http.ServeContent` for that response.
`ServeContent` would force status 200.

## Built-in 404 page

Title is the `NotFoundTitle` message.
Show the requested path.
Link to `/` with the `NotFoundBack` message.
Use the same stylesheet tag as the listing.
Load no JavaScript.

## Status codes

| Response | Status |
| --- | --- |
| Real file | 200 |
| Real file byte range | 206 |
| Real file, `If-None-Match` matches | 304 |
| Real file, `If-Match` fails | 412 |
| Real file, range outside the file | 416 |
| Listing | 200 |
| Directory index or site index as SPA miss | 200 |
| Site 404 | 404 |
| Built-in 404 page | 404 |
| Registered asset | 200 |
| Unregistered asset name | 404 |
| Asset URL, method not GET or HEAD | 405 |
| Directory without `/` | 308 |
| Method not GET or HEAD | 405 |

## Assets

The Sakura stylesheet is lewkit `sakuracss` 1.5.1.
Importing that package registers the file.
`NewApp` puts the asset handler in front of the file handler.

Do not vendor CSS in this module.
Do not fetch CSS at run time.

## Templates

Write pages in `*.templ`.
Commit the generated `*_templ.go` files.

```
//go:generate go tool templ generate
```

Add `github.com/a-h/templ/cmd/templ` as a Go tool in `go.mod`.
`mise run ci` runs `go generate ./...` before test and build.

## Language

Built-in pages and the 405 body use go-i18n.
Pick the language from the request `Accept-Language` header.
The query string is not used for language.
Do not add a `--lang` flag.
If the header is missing or matches no catalog, use English.

Message files live in `locales/active.*.toml` and are embedded.
The TOML files are the source of the strings.
Go looks up a message by ID. Do not keep a second English copy in Go.
This change ships English (`en`) and Portuguese (`pt`).
CLI help stays English.
Size units and timestamps stay as specified in this file.
Log lines stay English.

Add `github.com/nicksnyder/go-i18n/v2/goi18n` as a Go tool.

## Out of scope

- Upload, delete, mkdir, zip
- Auth and TLS
- Clean URLs (strip `.html`)
- Walk-up to a nested `index.html` on a miss
- Icon files
- Theme toggle JavaScript
