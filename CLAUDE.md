# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

`ctm` is a CLI tool for controlling [Time Machines Corporation](https://timemachinescorp.com) network clocks
over UDP (port 7372). It implements their Locator Protocol API, supporting both v1.x (35-byte responses)
and v2.0 (40-byte responses).

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
`up_set_time H:M:S:tenths:hundredths` (trailing components optional, but leading zeros required, e.g. `0:30`),
`down_run`, `down_pause`, `down_set_time H:M:S:tenths:hundredths` (same time syntax as `up_set_time`).

Global flags (may appear before or after the subcommand, but must precede
positional args): `-timeout duration` (default `2s`) and `-port` (default `7372`).
`ctm help`, `ctm -h`, and `ctm $SUBCOMMAND -h` print usage.

## Architecture

Everything lives in `main.go`. The code is organized around:

- **`ffcli` command tree** — `newCommandTree(config)` builds a root `ffcli.Command` with one
  subcommand per verb (called by `main()` and by the tests);
  `modeCommand` is the shared builder for simple mode switches, `setTimeCommand` for timer-set
  commands, and `statusCommand` wraps its bespoke logic
- **`clockConfig`** — shared `-timeout`/`-port` flag values, registered on the root and every
  subcommand flag set so flags work in either position; `addrport` joins host and port;
  the `dial` field is a test seam that swaps in a fake connection (production leaves it nil)
- **`locatorCommands` map** — string subcommand names to hex command bytes
- **`getStatus(dial, address, timeout)`** — sends a status query, decodes and prints the response
- **`sendCommand(dial, address, timeout, command)`** — sends a simple mode-switch command, expects an ACK
- **`sendSetCommand(dial, address, timeout, command, time)`** — sends a `SetTimer` struct for `up_set_time`/`down_set_time`
- **`extractTimePart(value, part)`** — parses a colon-delimited time string by index
  (components must be 0-255)

Wire format structs (all use `encoding/binary` with big-endian):

- `Response10` (34 bytes) — API v1.x status packet; the wire packet is 35 bytes
  (`api1PacketSize`), so one trailing byte is ignored as padding
- `Response20` (40 bytes, `api2PacketSize`) — API v2.0 status packet, fully decoded and printed by `status`
- `SetTimer` (6 bytes) — timer set command payload
- `Time10` / `Time20` — time display sub-fields (3 vs 4 bytes)

## Testing

`main_test.go` covers the wire structs, command bytes, time parsing, and subcommand
dispatch.  Dispatch tests drive the real `ffcli` tree via `newCommandTree` with a
`fakeConn` installed on `clockConfig.dial`; `dialClock` itself is tested against
loopback UDP sockets (happy path, timeout, bad address).  Coverage is ~81%.
`-h` cannot be tested in-process because the flag sets use `flag.ExitOnError`.

## Known Gaps

- API 2.0 features not implemented: dotmatrix text, relay, dimmer, RGB color, exec stored program
- UDP reads wait up to `-timeout` (default 2s); a silent clock exits with an error instead of hanging
