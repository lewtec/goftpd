# goftpd

A small HTTP file server. One binary. Point it at a directory and share files
on the local network.

Use it when you are in a lab with a slow uplink and you do not want to hand
out a USB stick. Compile it, copy the binary, run it.

By default it serves the current directory. Put the binary on a USB stick and
run it from there to share the whole stick.

Build:

```bash
mise exec -- go build -o goftpd ./cmd/goftpd
```

Ready-made binaries are on [GitHub Releases](https://github.com/lewtec/goftpd/releases)
(linux/mac/windows, amd64/arm64).

## Flags

| Flag | Default | Meaning |
| --- | --- | --- |
| `--addr` | `:8080` | Listen address |
| `--dir` / `-d` | `./` | Served directory |
| `--spa` | off | SPA mode. Directories never list. A directory with `index.html` serves that file. A miss serves `/404.html` (404), else `/index.html` (200), else the built-in 404 page. |

Without `--spa`, a directory shows a listing. A miss shows the built-in 404
page. User `index.html` and `404.html` files are ordinary files.

See [SPEC.md](SPEC.md) for the full request rules.

The listing and the built-in 404 page follow the browser `Accept-Language`
header. English and Portuguese ship in the binary. Missing or unknown
tags fall back to English.

Styles for the listing and the built-in 404 page live at `/__goftpd__/`.

## Release

[GoReleaser](https://goreleaser.com) + [svu](https://github.com/caarlos0/svu).
Archives and checksums only. Tags have no `v` prefix ([`.svu.yml`](.svu.yml)).

```bash
mise release          # next (svu) + goreleaser (needs GITHUB_TOKEN)
mise release patch    # or major | minor | next
```

CI: [`.github/workflows/autorelease.yml`](.github/workflows/autorelease.yml).
Push/PR runs `mise run ci`. GitHub Releases only via **Actions → Autorelease →
Run workflow** (`workflow_dispatch`).
