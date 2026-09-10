# Project Security Policy

## Confidential Disclosure

Please email [chicks.net@gmail.com](mailto:chicks.net@gmail.com) with a subject
line like "ctm SECURITY issue: $SUMMARY".  Details on when you experienced
the issue, logs, and other context are appreciated to assist with effective
triaging of your issue.

## Security Updates

End users should expect new releases to include any security updates and there
should be a notification in the release notes.
We may participate in other disclosure programs as circumstances may warrant.

## Known issues

There are no known security vulnerabilities in the software at the time
this was written.

## Security Risks

This tool talks to Time Machines Corporation clocks over UDP port 7372,
which is a protocol with no authentication or encryption.  Be aware of:

- Anyone on the network can send the same commands — the clock firmware
  accepts them from any source, so timers can be run, paused, or reset by
  anyone who can reach the device.
- Traffic is cleartext UDP, so commands and responses (including device
  name and MAC address) can be observed on the network.
- `ctm` does not verify that a response actually came from the intended
  clock, so a spoofer could feed it crafted (though harmless) output.

The practical mitigation is running `ctm` only on trusted, controlled
networks — which is how these clocks are normally deployed.
