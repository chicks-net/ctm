# ctm = Control Time Machines!

Control Time Machines (`ctm`) lets you control [timemachinescorp.com](https://timemachinescorp.com) clocks

Status: UDP communication is working.  Translating binary packet into golang struct is next job.

## Usage

I'd like for it to work like this:

```
ctm $CLOCK_IP
```

For now I'm testing with

```
go run main.go $CLOCK_IP
```

## References

* [TimeMachines' Locator Protocol API version 2.0](https://www.timemachinescorp.com/wp-content/uploads/TimeMachinesControlAPI.pdf)
* [the other repo on github for these clocks](https://github.com/ggmp3/Q-SYS-CSS-TimeMachines-Clock-B-Series-) - this looks pretty cool actually, but the language is obscure to me and I'm unlikely to license to devevelopment environment.
