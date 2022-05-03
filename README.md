# ctm = Control Time Machines!

Control Time Machines (`ctm`) lets you control [timemachinescorp.com](https://timemachinescorp.com) clocks

Status: Hacked to retrieve status from API1.1 clock, but working on how to update clock firmware without Windows.

## Usage

I'd like for it to work like this:

```
ctm $CLOCK_IP
```

For now I'm testing with

```
go run main.go $CLOCK_IP
```

Output looks like:

```
connected
sent query
read response
00000000  01 c0 a8 2a cc 70 b3 d5  75 68 e2 05 00 10 db 12  |...*.p..uh......|
00000010  0e 32 50 4f 45 5f 43 6c  6f 63 6b 5f 55 54 43 00  |.2POE_Clock_UTC.|
00000020  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
00000030  00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
Type 1
IP c0a82acc
MAC 70b3d57568e2
Ver 0500
Syncs 4315
Time {18 14 50}
Name POE_Clock_UTC
```

## References

* [TimeMachines' Locator Protocol API version 2.0](https://www.timemachinescorp.com/wp-content/uploads/TimeMachinesControlAPI.pdf)
* [the other repo on github for these clocks](https://github.com/ggmp3/Q-SYS-CSS-TimeMachines-Clock-B-Series-) - this looks pretty cool actually, but the language is obscure to me and I'm unlikely to license to devevelopment environment.
