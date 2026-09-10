package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type IPAddr [4]byte

// API 1.x structures
type Time10 struct {
	Hour   uint8
	Minute uint8
	Second uint8
}

type Response10 struct {
	DeviceType   uint8
	ClientIP     IPAddr
	MACAddress   [6]uint8
	FirmwareVer  [2]uint8
	NTPSyncCount uint16
	DisplayTime  Time10
	DeviceName   [16]uint8
}

// API 2.x structures
type Time20 struct {
	Hour   uint8
	Minute uint8
	Second uint8
	Tenths uint8
}

type Response20 struct {
	DeviceType   uint8
	ClientIP     IPAddr
	MACAddress   [6]uint8
	FirmwareVer  [2]uint8
	NTPSyncCount uint16
	DisplayTime  Time20
	DisplayMode  uint8
	DownAlarm    uint8
	Days         uint8
	Digits       uint8
	WifiSignal   uint8
	DeviceName   [16]uint8
}

// Structures for API1.x and API2.x

type SetTimer struct {
	Command    uint8
	Hour       uint8
	Minute     uint8
	Second     uint8
	Tenths     uint8
	Hundredths uint8
}

const maxBufferSize = 48 // the biggest response packet is 40 bytes

const (
	defaultTimeout = 2 * time.Second // how long to wait for a clock to respond
	defaultPort    = 7372            // UDP port the clocks listen on
)

const (
	api1PacketSize = 35 // API 1.x status packet: 34-byte struct plus one trailing pad byte
	api2PacketSize = 40 // API 2.0 status packet
	ackPacketSize  = 2  // command acknowledgement: "A" plus one status byte
)

// locatorCommands maps internal command names to the raw bytes the
// Locator Protocol expects on the wire
var locatorCommands = map[string]string{
	"device_query":    "\xa1\x04\xb2",
	"up_mode_ms":      "\xa2\x00\x00",
	"up_mode_hms":     "\xa2\x01\x00",
	"up_mode_pause":   "\xa3\x00\x00",
	"up_mode_run":     "\xa3\x01\x00",
	"up_reset_ms":     "\xa4\x00\x00",
	"up_reset_hms":    "\xa4\x01\x00",
	"down_mode_pause": "\xa6\x00\x00",
	"down_mode_run":   "\xa6\x01\x00",
	"time_mode":       "\xa8\x01\x00",
	"up_set_time":     "\xaa",
	"down_set_time":   "\xab",
}

// utility functions - type conversion and defaults
func (ip IPAddr) String() string {
	return fmt.Sprintf("%v.%v.%v.%v", int(ip[0]), int(ip[1]), int(ip[2]), int(ip[3]))
}

// clockConfig holds the shared command-line settings that every
// subcommand accepts.  The same config is bound to the root flag set and
// to every subcommand flag set, so -timeout and -port may be spelled
// either before or after the subcommand name.  dial exists for tests,
// which swap it for a fake; production code leaves it nil.
type clockConfig struct {
	timeout time.Duration
	port    int
	dial    dialer
}

// register binds the shared -timeout and -port flags onto a flag set
func (c *clockConfig) register(fs *flag.FlagSet) {
	fs.DurationVar(&c.timeout, "timeout", defaultTimeout, "how long to wait for a clock to respond")
	fs.IntVar(&c.port, "port", defaultPort, "UDP port the clock listens on")
}

// addrport joins a clock host with the configured port
func (c *clockConfig) addrport(host string) string {
	return net.JoinHostPort(host, strconv.Itoa(c.port))
}

// convert a failed clock read into a clear error, calling out the
// timeout case so a silent clock is not mistaken for a crash
func readError(address string, timeout time.Duration, err error) error {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("clock at %s did not respond (timeout after %s)", address, timeout)
	}
	return fmt.Errorf("reading response from %s: %w", address, err)
}

// dialer is the seam the tests use to stand in for real clock
// connections: production code passes dialClock, tests pass a fake
type dialer func(address string, timeout time.Duration) (net.Conn, error)

// dial the clock and arm a read deadline so a silent clock cannot
// block the caller forever
func dialClock(address string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", address, err)
	}
	err = conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting read deadline on %s: %w", address, err)
	}
	return conn, nil
}

// displayModeString turns an API 2.0 display-mode byte into a
// human-readable description: the low 3 bits pick the mode (time,
// up timer, down timer, interval counters), bit 6 (0x40) says
// whether the timer is running and bits 7/5 (0x80/0x20) pick the
// display format
func displayModeString(mode uint8) string {
	var name string
	switch mode & 0x07 {
	case 0:
		return "time"
	case 1:
		name = "up timer"
	case 2:
		name = "down timer"
	case 3:
		name = "interval count up"
	case 4:
		name = "interval count down"
	default:
		name = fmt.Sprintf("unknown (0x%02x)", mode&0x07)
	}

	parts := []string{name}
	if mode&0x40 == 0x40 {
		parts = append(parts, "running")
	} else {
		parts = append(parts, "stopped")
	}
	if mode&0x80 == 0x80 {
		parts = append(parts, "M:S.Tenths")
	} else if mode&0x20 == 0x20 {
		parts = append(parts, "D:H:M")
	} else {
		parts = append(parts, "H:M:S")
	}
	return strings.Join(parts, ", ")
}

// dialOrTest dials with the config's dialer when a test has
// installed one, and with the real dialer otherwise
func (c *clockConfig) dialOrTest(address string, timeout time.Duration) (net.Conn, error) {
	if c.dial != nil {
		return c.dial(address, timeout)
	}
	return dialClock(address, timeout)
}

// functions that talk to the clock
func getStatus(dial dialer, address string, timeout time.Duration) error {
	conn, err := dial(address, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = fmt.Fprint(conn, locatorCommands["device_query"])
	if err != nil {
		return fmt.Errorf("sending status query to %s: %w", address, err)
	}
	fmt.Printf("sent status query to %s\n", address)

	response := make([]byte, maxBufferSize) // buffer for UDP responses
	packetLen, err := conn.Read(response)
	if err == nil {
		fmt.Println("response hexdump:")
		fmt.Printf("%s", hex.Dump(response[:packetLen]))

		if packetLen == api1PacketSize {
			// API version 1.x
			fmt.Printf("packet length %d (API version 1.x)\n", packetLen)
			resp10 := Response10{}
			buf := bytes.NewReader(response)
			err = binary.Read(buf, binary.BigEndian, &resp10)
			if err != nil {
				return fmt.Errorf("decoding %d-byte response as API 1.x: %w", packetLen, err)
			}

			fmt.Printf("Type %x\n", resp10.DeviceType)
			fmt.Printf("IP %v\n", resp10.ClientIP)
			fmt.Printf("MAC %x\n", resp10.MACAddress)
			fmt.Printf("Ver %x\n", resp10.FirmwareVer)
			fmt.Printf("Syncs %d\n", resp10.NTPSyncCount)
			fmt.Printf("Time %d\n", resp10.DisplayTime)
			fmt.Printf("Name %s\n", resp10.DeviceName)
		} else if packetLen == api2PacketSize {
			// API version 2.0
			fmt.Printf("packet length %d (API version 2.0)\n", packetLen)
			resp20 := Response20{}
			buf := bytes.NewReader(response)
			err = binary.Read(buf, binary.BigEndian, &resp20)
			if err != nil {
				return fmt.Errorf("decoding %d-byte response as API 2.0: %w", packetLen, err)
			}

			fmt.Printf("Type %x\n", resp20.DeviceType)
			fmt.Printf("IP %v\n", resp20.ClientIP)
			fmt.Printf("MAC %x\n", resp20.MACAddress)
			fmt.Printf("Ver %x\n", resp20.FirmwareVer)
			fmt.Printf("Syncs %d\n", resp20.NTPSyncCount)
			fmt.Printf("Time %d:%d:%d.%d\n", resp20.DisplayTime.Hour,
				resp20.DisplayTime.Minute,
				resp20.DisplayTime.Second,
				resp20.DisplayTime.Tenths)
			fmt.Printf("Mode %s\n", displayModeString(resp20.DisplayMode))
			if resp20.DownAlarm&0x80 == 0x80 {
				fmt.Printf("Down alarm on (%d seconds)\n", resp20.DownAlarm&0x7f)
			} else {
				fmt.Println("Down alarm off")
			}
			fmt.Printf("Days %d\n", uint16(resp20.Days)<<3|uint16(resp20.Digits>>5))
			switch resp20.Digits & 0x1f {
			case 0:
				fmt.Println("Digits 4/6")
			case 1:
				fmt.Println("Digits (D):H:M:S")
			case 2:
				fmt.Println("Digits (H):M:S.Tenths")
			default:
				fmt.Printf("Digits unknown (%d)\n", resp20.Digits&0x1f)
			}
			if resp20.WifiSignal == 0 {
				fmt.Println("Wifi wired")
			} else {
				fmt.Printf("Wifi -%d dBm\n", resp20.WifiSignal)
			}
			fmt.Printf("Name %s\n", resp20.DeviceName)
		} else {
			fmt.Printf("packet length %d\n", packetLen)
			return fmt.Errorf("unexpected number of bytes returned so we don't know which protocol it is talking")
		}
	} else {
		return readError(address, timeout, err)
	}
	return nil
}

func sendCommand(dial dialer, address string, timeout time.Duration, command string) error {
	conn, err := dial(address, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = fmt.Fprint(conn, locatorCommands[command])
	if err != nil {
		return fmt.Errorf("sending command %s to %s: %w", command, address, err)
	}
	fmt.Printf("sent command %s to %s\n", command, address)

	response := make([]byte, maxBufferSize) // buffer for UDP responses
	packetLen, err := conn.Read(response)
	if err == nil {
		if packetLen != ackPacketSize {
			fmt.Printf("packet length %d\n", packetLen)
			return fmt.Errorf("unexpected packet size in UDP response")
		}
		if string(response[0]) != "A" {
			fmt.Println("response hexdump:")
			fmt.Printf("%s", hex.Dump(response[:packetLen]))
			return fmt.Errorf("response does not look like an acknowledgement")
		}
		fmt.Println("acked by clock")
	} else {
		return readError(address, timeout, err)
	}
	return nil
}

func extractTimePart(value string, part int) (uint8, error) {
	parts := strings.Split(value, ":")

	if len(parts) > part {
		n, err := strconv.Atoi(parts[part])
		if err != nil {
			return 0, fmt.Errorf("parsing %q as a time component: %w", parts[part], err)
		}
		if n < 0 || n > 255 {
			return 0, fmt.Errorf("time component %q out of range (must be 0-255)", parts[part])
		}
		return uint8(n), nil
	}
	return uint8(0), nil
}

func sendSetCommand(dial dialer, address string, timeout time.Duration, command string, timeSpec string) error {
	var err error
	timer := SetTimer{}
	timer.Command = uint8(locatorCommands[command][0])

	timer.Hour, err = extractTimePart(timeSpec, 0)
	if err != nil {
		return err
	}
	timer.Minute, err = extractTimePart(timeSpec, 1)
	if err != nil {
		return err
	}
	timer.Second, err = extractTimePart(timeSpec, 2)
	if err != nil {
		return err
	}
	timer.Tenths, err = extractTimePart(timeSpec, 3)
	if err != nil {
		return err
	}
	timer.Hundredths, err = extractTimePart(timeSpec, 4)
	if err != nil {
		return err
	}

	fmt.Println(timer)

	conn, err := dial(address, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	var payload bytes.Buffer // buffer for UDP send
	err = binary.Write(&payload, binary.BigEndian, timer)
	if err != nil {
		return fmt.Errorf("encoding SetTimer struct: %w", err)
	}

	length, err := conn.Write(payload.Bytes())
	if err != nil {
		return fmt.Errorf("sending command %s to %s: %w", command, address, err)
	}
	fmt.Printf("sent command %s to %s (%d bytes)\n", command, address, length)

	response := make([]byte, maxBufferSize) // buffer for UDP responses
	packetLen, err := conn.Read(response)
	if err == nil {
		if packetLen != ackPacketSize {
			fmt.Printf("packet length %d\n", packetLen)
			return fmt.Errorf("unexpected packet size in UDP response")
		}
		if string(response[0]) != "A" {
			fmt.Println("response hexdump:")
			fmt.Printf("%s", hex.Dump(response[:packetLen]))
			return fmt.Errorf("response does not look like an acknowledgement")
		}
		fmt.Println("acked by clock")
	} else {
		return readError(address, timeout, err)
	}
	return nil
}

// functions that build the command-line interface

// commandFlagSet builds the flag set for a subcommand, carrying the
// shared clock flags
func commandFlagSet(cfg *clockConfig, name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	cfg.register(fs)
	return fs
}

// requireAddress validates that a subcommand got exactly one
// positional argument - the clock address
func requireAddress(name string, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%s requires exactly 1 argument (the clock address), got %d", name, len(args))
	}
	return args[0], nil
}

// statusCommand builds the subcommand that queries a clock's status
func statusCommand(cfg *clockConfig) *ffcli.Command {
	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "ctm status [flags] <address>",
		ShortHelp:  "query the clock and print its status",
		FlagSet:    commandFlagSet(cfg, "status"),
		Exec: func(_ context.Context, args []string) error {
			address, err := requireAddress("status", args)
			if err != nil {
				return err
			}
			return getStatus(cfg.dialOrTest, cfg.addrport(address), cfg.timeout)
		},
	}
}

// modeCommand builds a subcommand that sends one mode-switch command
// to the clock at the given address argument
func modeCommand(cfg *clockConfig, name string, command string, help string) *ffcli.Command {
	return &ffcli.Command{
		Name:       name,
		ShortUsage: "ctm " + name + " [flags] <address>",
		ShortHelp:  help,
		FlagSet:    commandFlagSet(cfg, name),
		Exec: func(_ context.Context, args []string) error {
			address, err := requireAddress(name, args)
			if err != nil {
				return err
			}
			return sendCommand(cfg.dialOrTest, cfg.addrport(address), cfg.timeout, command)
		},
	}
}

// setTimeCommand builds a subcommand that sends a SetTimer struct
// for the named timer command (up_set_time or down_set_time)
func setTimeCommand(cfg *clockConfig, name string, command string, help string) *ffcli.Command {
	return &ffcli.Command{
		Name:       name,
		ShortUsage: "ctm " + name + " [flags] <address> H:M:S:tenths:hundredths",
		ShortHelp:  help,
		FlagSet:    commandFlagSet(cfg, name),
		Exec: func(_ context.Context, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("%s requires exactly 2 arguments (clock address and H:M:S time), got %d", name, len(args))
			}
			return sendSetCommand(cfg.dialOrTest, cfg.addrport(args[0]), cfg.timeout, command, args[1])
		},
	}
}

// newCommandTree builds the full ctm command tree around the given
// config.  main() and the tests share this so the dispatch logic is
// exercised exactly as shipped.
func newCommandTree(config *clockConfig) *ffcli.Command {
	rootFlags := flag.NewFlagSet("ctm", flag.ExitOnError)
	config.register(rootFlags)

	root := &ffcli.Command{
		Name:       "ctm",
		ShortUsage: "ctm [flags] <subcommand> [flags] <address> ...",
		ShortHelp:  "control Time Machines Corp. network clocks over UDP",
		FlagSet:    rootFlags,
		Subcommands: []*ffcli.Command{
			statusCommand(config),
			modeCommand(config, "time", "time_mode", "put the clock into time display mode"),
			modeCommand(config, "up_ms", "up_mode_ms", "put the uptimer in minutes:seconds mode"),
			modeCommand(config, "up_hms", "up_mode_hms", "put the uptimer in hours:minutes:seconds mode"),
			modeCommand(config, "up_run", "up_mode_run", "run the uptimer"),
			modeCommand(config, "up_pause", "up_mode_pause", "pause the uptimer"),
			modeCommand(config, "up_reset_ms", "up_reset_ms", "reset the uptimer in minutes:seconds mode"),
			modeCommand(config, "up_reset_hms", "up_reset_hms", "reset the uptimer in hours:minutes:seconds mode"),
			setTimeCommand(config, "up_set_time", "up_set_time", "set the uptimer to H:M:S:tenths:hundredths (smaller units optional)"),
			modeCommand(config, "down_run", "down_mode_run", "run the downtimer"),
			modeCommand(config, "down_pause", "down_mode_pause", "pause the downtimer"),
			setTimeCommand(config, "down_set_time", "down_set_time", "set the downtimer to H:M:S:tenths:hundredths (smaller units optional)"),
		},
		Exec: func(_ context.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown subcommand %q (run 'ctm help' to see the subcommands)", args[0])
			}
			return flag.ErrHelp
		},
	}

	root.Subcommands = append(root.Subcommands, &ffcli.Command{
		Name:       "help",
		ShortUsage: "ctm help",
		ShortHelp:  "show help for ctm",
		FlagSet:    flag.NewFlagSet("help", flag.ExitOnError),
		Exec: func(_ context.Context, _ []string) error {
			fmt.Print(ffcli.DefaultUsageFunc(root))
			return nil
		},
	})

	return root
}

func main() {
	var config clockConfig

	root := newCommandTree(&config)

	err := root.ParseAndRun(context.Background(), os.Args[1:])
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(os.Stderr, "ctm: %v\n", err)
		}
		os.Exit(1)
	}
}
