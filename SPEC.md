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
| `/__goftpd__/` | reserved prefix | internal path, magic path |
| `index.html` in a directory | directory index | homepage, entry |
| `/index.html` at the served directory | site index | root index |
| `/404.html` at the served directory | site 404 | custom 404, error page |
| Templ HTML from goftpd | built-in page | internal template, our 404 |
| `a-h/templ` generated Go | generated file | compiled template |

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--addr` | `:8080` | Listen address |
| `--dir` | `./` | Served directory |
| `--spa` | off | SPA mode |

All user-facing strings are English.
Do not add go-i18n in this change.

## Request path

Use `r.URL.Path`.
Do not use `r.RequestURI`.
The query string is not part of the file path.

Decode `%xx` escapes.
Reject a path that escapes the served directory.
Open files through `os.Root` on the served directory.

Honor `GET` and `HEAD`.
For other methods, return `405`.

## Reserved prefix

Paths `/__goftpd__` and `/__goftpd__/*` never read the served directory.
They never enter SPA mode.
They never produce a listing.

Map URL to embed files:

| URL | File in the module |
| --- | --- |
| `/__goftpd__/sakura.css` | `assets/sakura.css` |
| `/__goftpd__/sakura-dark.css` | `assets/sakura-dark.css` |
| `/__goftpd__/listing.css` | `assets/listing.css` |

A later change may add `/__goftpd__/icons/*` from `assets/icons/`.
Do not add icon files in this change.

If the embed file is missing, return the built-in 404 page with status 404.
Do not list the embed file system.

A directory named `__goftpd__` in the served directory is not reachable.

## Trailing slash

If the path names a directory and has no final `/`, send `308`
to the same path with `/`.
Do not redirect a miss.
Do not redirect a file.

## Real file

If the path names a regular file, serve that file with status 200.
Use `http.ServeContent` so `HEAD` and range requests work.
Set `Cache-Control: max-age=5` before you write the body.

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
Each crumb is a link.

The listing is server-rendered HTML from templ.
The page loads no JavaScript.

Stylesheets:

```html
<link rel="stylesheet" href="/__goftpd__/sakura.css" media="screen">
<link rel="stylesheet" href="/__goftpd__/sakura-dark.css" media="screen and (prefers-color-scheme: dark)">
<link rel="stylesheet" href="/__goftpd__/listing.css">
```

`listing.css` sets `body { max-width: none; }` so a table can use the
screen width.
Sakura 1.5.1 sets `max-width: 38em` on `body`.
That width is for articles, not for a file table.

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

English title `Not found`.
Show the requested path.
Link to `/`.
Use the same stylesheets as the listing.
Load no JavaScript.

## Status codes

| Response | Status |
| --- | --- |
| Real file | 200 |
| Listing | 200 |
| Directory index or site index as SPA miss | 200 |
| Site 404 | 404 |
| Built-in 404 page | 404 |
| Directory without `/` | 308 |
| Method not GET or HEAD | 405 |

## Assets

Vendor sakura.css 1.5.1 and sakura-dark.css 1.5.1 in `assets/`.
Put the version in a comment at the top of each file.
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

Rewrite Portuguese CLI text, log lines, errors, and README to English
in this change.
Keep go-i18n for a later change.

## Out of scope

- Upload, delete, mkdir, zip
- Auth and TLS
- Clean URLs (strip `.html`)
- Walk-up to a nested `index.html` on a miss
- Icon files
- Theme toggle JavaScript
- Accept-Language
