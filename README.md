# ctm = Control Time Machines!

Control Time Machines (`ctm`) lets you control [timemachinescorp.com](https://timemachinescorp.com) clocks

Status: Can retrieve status from API1.1 clock.  Next step is to implement subcommands and timer control sequences.

## Usage

I'd like for it to work like this:

```
ctm $CLOCK_IP
```

For now I'm testing with

```
go run main.go status $CLOCK_IP
```

Output looks like:

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

## Known bugs

* There is no timeout yet so you will need to hit <kbd>Ctrl</kbd>-<kbd>C</kbd> to exit if you put in an invalid host or IP.

## References

* [TimeMachines' Locator Protocol API version 2.0](https://www.timemachinescorp.com/wp-content/uploads/TimeMachinesControlAPI.pdf)
* I don't have a URL for the API v1.1 docs, but TimeMachinesCorp sent me a copy and not much has changed.
* [the other repo on github for these clocks](https://github.com/ggmp3/Q-SYS-CSS-TimeMachines-Clock-B-Series-) - this looks pretty cool actually, but the language is obscure to me and I'm unlikely to license the development environment.
