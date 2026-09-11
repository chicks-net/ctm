<!-- markdownlint-disable MD033 -->
# ctm = Control Time Machines

[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/chicks-net/ctm/badge)](https://scorecard.dev/viewer/?uri=github.com/chicks-net/ctm)

Control Time Machines (`ctm`) lets you control clocks
from [Time Machines Corp.](https://timemachinescorp.com).

Status: Can retrieve status, use uptimers and downtimers with an
API1.1 or API2.0 clock.

## Install

I'd like to get this in homebrew and Fedora repos eventually, but for now you
can build the program yourself.  In the main repo directory either of these
will build `ctm` for your platform:

```bash
just build   # via the justfile (also: just test, just fmt)
make         # via the Makefile
```

## Development prerequisites

The `just` recipes need a few tools installed: `just`, `gh`,
`shellcheck`, `markdownlint-cli2`, `jq`, `gum`, `cue`, and the `gh-observer`
gh extension.  On macOS with Homebrew, `./.just/lib/install-prerequisites.sh`
checks for all of them and installs anything missing - run it once and you're
set.  The `cue` CLI is new for contributors: it validates `.repo.toml` against
`docs/repo-toml.cue` (`just cue-verify`).

## Usage

Invocation:

```bash
ctm [flags] $SUBCOMMAND [flags] $CLOCK_IP
```

Global flags (may appear before or after the subcommand, but must precede
positional args):

- `-timeout duration` - how long to wait for a clock to respond
  (default `2s`)
- `-port` - UDP port the clock listens on (default `7372`)

`ctm help` (or `ctm -h`, or `ctm $SUBCOMMAND -h`) prints usage.

Subcommands:

- `status` returns the clock's status
- `time` puts the clock into time display mode
- `up_ms` puts the clock into uptimer mode displaying minutes and seconds
- `up_hms` puts the clock into uptimer mode displaying hours, minutes and seconds
- `up_run` tells the uptimer to run
- `up_pause` tells the uptimer to pause
- `up_reset_ms` resets the uptimer displaying minutes and seconds
- `up_reset_hms` resets the uptimer displaying hours, minutes and seconds
- `up_set_time H:M:S:tenths:hundreds` sets the uptimer to the
  hours`:`minutes`:`seconds`:`tenths`:`hundreths and you can drop off any of
  the smaller units that are irrelevant to you.  If you want no hours you still
  need to start with `0:`.
- `down_run` tells the downtimer to run (API2.0 clocks only)
- `down_pause` tells the downtimer to pause (API2.0 clocks only)
- `down_set_time H:M:S:tenths:hundreds` sets the downtimer like
  `up_set_time` sets the uptimer (API2.0 clocks only)

Status output looks like:

```ScreenOutput
sent status query to 192.168.42.204:7372
response hexdump:
00000000  01 c0 a8 2a cc 70 b3 d5  75 68 e2 05 00 12 2e 02  |...*.p..uh......|
00000010  2a 2c 50 4f 45 5f 43 6c  6f 63 6b 5f 55 54 43 00  |*,POE_Clock_UTC.|
00000020  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
packet length 35 (API version 1.x)
Type 1
IP 192.168.42.204
MAC 70b3d57568e2
Ver 0500
Syncs 4654
Time {2 42 44}
Name POE_Clock_UTC
```

Using the uptimer looks like:

```bash
$ ./ctm up_ms 192.168.42.204
sent command up_mode_ms to 192.168.42.204:7372
acked by clock
$ ./ctm up_run 192.168.42.204
sent command up_mode_run to 192.168.42.204:7372
acked by clock
$ ./ctm time 192.168.42.204
sent command time_mode to 192.168.42.204:7372
acked by clock
```

## Known bugs

- Coming soon:
  - binary releases like <https://github.com/fini-net/gh-observer/blob/main/.github/workflows/release.yml>
  - [signed releases](https://github.com/chicks-net/ctm/issues/42)
  - [dimmer set (API2.0)](https://github.com/chicks-net/ctm/issues/40)
  - [color set for RGB (API2.0)](https://github.com/chicks-net/ctm/issues/41)
  - [Homebrew install](https://github.com/chicks-net/homebrew-chicks/issues/74)
- Unimplemented:
  - setting dotmatrix text - I don't have [a device to test this](https://timemachinescorp.com/timezone-dot-matrix-network-clocks/) with yet.
  - exec stored program (API2.0)
  - relay close (API2.0)

## References

- TimeMachinesCorp API documentation:
  - [Locator Protocol API v2.0](https://www.timemachinescorp.com/wp-content/uploads/TimeMachinesControlAPI.pdf)
- I don't have a URL for the API v1.1 docs, but TimeMachinesCorp sent me a copy
  and not much has changed.
- [Glen Gorton's project](https://github.com/ggmp3/Q-SYS-CSS-TimeMachines-Clock-B-Series-)
  is the other repo on github for these clicks.  It looks pretty cool actually,
  but the language is obscure to me and I'm unlikely to license the development
  environment just to try this out.
