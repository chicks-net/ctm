# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

`ctm` is a CLI tool for controlling [Time Machines Corporation](https://timemachinescorp.com) network clocks
over UDP (port 7372). It implements their Locator Protocol API, supporting both v1.x (35-byte responses)
and v2.0 (40-byte responses, currently unimplemented beyond detection).

## Build and Run

```sh
make          # builds ./ctm binary
go fmt main.go  # format code
```

No external dependencies — standard library only.

## Usage

```sh
ctm $SUBCOMMAND $CLOCK_IP
```

Subcommands: `status`, `time`, `up_ms`, `up_hms`, `up_run`, `up_pause`, `up_reset_ms`, `up_reset_hms`,
`up_set_time H:M:S:tenths:hundredths` (trailing components optional, but leading zeros required, e.g. `0:30`).

## Architecture

Everything lives in `main.go`. The code is organized around:

- **`locator_commands` map** — string subcommand names to hex command bytes
- **`get_status(address)`** — sends a status query, decodes and prints the response
- **`send_command(address, command)`** — sends a simple mode-switch command, expects an ACK
- **`send_set_command(address, command, time)`** — sends a `SetTimer` struct for `up_set_time`
- **`extract_time_part(time, part)`** — parses a colon-delimited time string by index

Wire format structs (all use `encoding/binary` with big-endian):

- `Response10` (35 bytes) — API v1.x status packet
- `Response20` (40 bytes) — API v2.0 status packet (parsed but features unimplemented)
- `SetTimer` (6 bytes) — timer set command payload
- `Time10` / `Time20` — time display sub-fields (3 vs 4 bytes)

## Known Gaps

- No UDP timeout — invalid hosts require Ctrl-C to exit
- API 2.0 features panic if encountered: downtimers, dotmatrix text, relay, dimmer, RGB color, exec stored program
