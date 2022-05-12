# ctm = Control Time Machines!

Control Time Machines (`ctm`) lets you control [timemachinescorp.com](https://timemachinescorp.com) clocks

Status: Can retrieve status, use uptimers and go back to time mode with an API1.1 clock.

## Install

I'd like to get this in homebrew and Fedora repos eventually, but for now there is a `Makefile`
that will build the program for you.  So in the main repo directory you should be able to run `make`
and get a build of `ctm` for your platform.

## Usage

Invocation:

```
ctm $SUBCOMMAND $CLOCK_IP
```

Subcommands:
* `status` returns the clock's status
* `time` puts the clock into time display mode
* `up_ms` puts the clock into uptimer mode displaying minutes and seconds
* `up_hms` puts the clock into uptimer mode displaying hours, minutes and seconds
* `up_run` tells the uptimer to run
* `up_pause` tells the uptimer to pause
* `up_reset_ms` resets the uptimer displaying minutes and seconds
* `up_reset_hms` resets the uptimer displaying hours, minutes and seconds
* `up_set_time H:M:S:tenths:hundreds` sets the uptimer to the hours`:`minutes`:`seconds`:`tenths`:`hundreths and you can drop off any of the smaller units that are irrelevant to you.  If you want no hours you still need to start with `0:`.

Status output looks like:

```
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

```
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

* There is no timeout yet so you will need to hit <kbd>Ctrl</kbd>-<kbd>C</kbd> to exit if you put in an invalid host or IP.
* Unimplemented:
    * downtimers - doable but not coded yet
    * setting dotmatrix text - I don't have a device to test with
    * setting timers while running - doable but not coded yet
    * exec stored program (API2.0)
    * relay close (API2.0)
    * dimmer set (API2.0)
    * color set for RGB (API2.0)

## References

* [TimeMachines' Locator Protocol API version 2.0](https://www.timemachinescorp.com/wp-content/uploads/TimeMachinesControlAPI.pdf)
* I don't have a URL for the API v1.1 docs, but TimeMachinesCorp sent me a copy and not much has changed.
* [the other repo on github for these clocks](https://github.com/ggmp3/Q-SYS-CSS-TimeMachines-Clock-B-Series-) - this looks pretty cool actually, but the language is obscure to me and I'm unlikely to license the development environment.
