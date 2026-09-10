# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

`ctm` is a CLI tool for controlling [Time Machines Corporation](https://timemachinescorp.com) network clocks
over UDP (port 7372). It implements their Locator Protocol API, supporting both v1.x (35-byte responses)
and v2.0 (40-byte responses, currently unimplemented beyond detection).

## Build and Run

```sh
just build    # builds ./ctm via `go build -o ctm .` (preferred)
make          # same build via the Makefile
just test     # go test ./...
just fmt      # reports files needing gofmt
go fmt .      # format code
```

Module path is `github.com/chicks-net/ctm`, pinned to Go 1.25 (`go.mod`).
One external dependency: `github.com/peterbourgon/ff/v3` (Apache-2.0, zero
transitive deps) — used for the `ffcli` subcommand tree; taken on for
issue #16 because it wraps stdlib `flag.FlagSet` with a small API surface
(help/usage generation, subcommand dispatch) without the weight of
cobra/urfave.  Flag parsing is flags-first: `-timeout` and `-port` may
appear before or after the subcommand, but must precede positional args.

The root `justfile` imports recipe modules from `.just/`:
`compliance.just`, `gh-process.just` (PR lifecycle), `pr-hook.just`,
`shellcheck.just`, `cue-verify.just`, `claude.just`, `copilot.just`,
`repo-toml.just`, `template-sync.just` (template updates).  Run `just`
or `just --list` to see all recipes.
`.repo.toml` holds repo metadata (description/topics/license/feature flags),
validated by `cue vet .repo.toml docs/repo-toml.cue` via `just cue-verify`.
`.just/repo-toml.sh` is generated (gitignored) by `just repo_toml_generate`.

The `.just/` modules track the FINI template-repo via `.just/CHECKSUMS.json`.
`just checksums_verify` reports drift from template versions and
`just update_from_template` safely pulls updates (locally modified files
are skipped, never clobbered).

## Usage

```sh
ctm [flags] $SUBCOMMAND [flags] $CLOCK_IP
```

Subcommands: `status`, `time`, `up_ms`, `up_hms`, `up_run`, `up_pause`, `up_reset_ms`, `up_reset_hms`,
`up_set_time H:M:S:tenths:hundredths` (trailing components optional, but leading zeros required, e.g. `0:30`).

Global flags (may appear before or after the subcommand, but must precede
positional args): `-timeout duration` (default `2s`) and `-port` (default `7372`).
`ctm help`, `ctm -h`, and `ctm $SUBCOMMAND -h` print usage.

## Architecture

Everything lives in `main.go`. The code is organized around:

- **`ffcli` command tree** — `main()` builds a root `ffcli.Command` with one subcommand per verb;
  `mode_command` is the shared builder for simple mode switches, `status_command` and `up_set_time_command`
  wrap their bespoke logic
- **`clockConfig`** — shared `-timeout`/`-port` flag values, registered on the root and every
  subcommand flag set so flags work in either position; `addrport` joins host and port
- **`locator_commands` map** — string subcommand names to hex command bytes
- **`get_status(address, timeout)`** — sends a status query, decodes and prints the response
- **`send_command(address, timeout, command)`** — sends a simple mode-switch command, expects an ACK
- **`send_set_command(address, timeout, command, time)`** — sends a `SetTimer` struct for `up_set_time`
- **`extract_time_part(time, part)`** — parses a colon-delimited time string by index

Wire format structs (all use `encoding/binary` with big-endian):

- `Response10` (35 bytes) — API v1.x status packet
- `Response20` (40 bytes) — API v2.0 status packet (parsed but features unimplemented)
- `SetTimer` (6 bytes) — timer set command payload
- `Time10` / `Time20` — time display sub-fields (3 vs 4 bytes)

## Known Gaps

- API 2.0 features return a "not implemented" error if encountered: downtimers, dotmatrix text, relay, dimmer, RGB color, exec stored program
- Downtimer subcommands (`down_run`, `down_pause`, `down_set_time`) not yet implemented (tracked: [issue #3](https://github.com/chicks-net/ctm/issues/3))
- UDP reads wait up to `-timeout` (default 2s); a silent clock exits with an error instead of hanging
