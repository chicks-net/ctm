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
	DeviceType  uint8
	ClientIP    IPAddr
	MAC_address [6]uint8
	FirmwareVer [2]uint8
	NTPSyncCnt  uint16
	DisplayTime Time10
	DeviceName  [16]uint8
}

// API 2.x structures
type Time20 struct {
	Hour   uint8
	Minute uint8
	Second uint8
	Tenths uint8
}

type Response20 struct {
	DeviceType  uint8
	ClientIP    IPAddr
	MAC_address [6]uint8
	FirmwareVer [2]uint8
	NTPSyncCnt  uint16
	DisplayTime Time20
	DisplayMode uint8
	DownAlarm   uint8
	Days        uint8
	Digits      uint8
	WifiSignal  uint8
	DeviceName  [16]uint8
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

var (
	locator_commands = make(map[string]string)
)

func init() {
	locator_commands["device_query"] = "\xa1\x04\xb2"
	locator_commands["up_mode_ms"] = "\xa2\x00\x00"
	locator_commands["up_mode_hms"] = "\xa2\x01\x00"
	locator_commands["up_mode_pause"] = "\xa3\x00\x00"
	locator_commands["up_mode_run"] = "\xa3\x01\x00"
	locator_commands["up_reset_ms"] = "\xa4\x00\x00"
	locator_commands["up_reset_hms"] = "\xa4\x01\x00"
	locator_commands["down_mode_pause"] = "\xa6\x00\x00"
	locator_commands["down_mode_run"] = "\xa6\x01\x00"
	locator_commands["time_mode"] = "\xa8\x01\x00"
	locator_commands["up_set_time"] = "\xaa"
	locator_commands["down_set_time"] = "\xab"
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
func read_error(address string, timeout time.Duration, err error) error {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("clock at %s did not respond (timeout after %s)", address, timeout)
	}
	return fmt.Errorf("reading response from %s: %w", address, err)
}

// dialer is the seam the tests use to stand in for real clock
// connections: production code passes dial_clock, tests pass a fake
type dialer func(address string, timeout time.Duration) (net.Conn, error)

// dial the clock and arm a read deadline so a silent clock cannot
// block the caller forever
func dial_clock(address string, timeout time.Duration) (net.Conn, error) {
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

// display_mode_string turns an API 2.0 display-mode byte into a
// human-readable description: the low 3 bits pick the mode (time,
// up timer, down timer, interval counters), bit 6 (0x40) says
// whether the timer is running and bits 7/5 (0x80/0x20) pick the
// display format
func display_mode_string(mode uint8) string {
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

// dial_clock_or_test dials with the config's dialer when a test has
// installed one, and with the real dialer otherwise
func (c *clockConfig) dial_clock_or_test(address string, timeout time.Duration) (net.Conn, error) {
	if c.dial != nil {
		return c.dial(address, timeout)
	}
	return dial_clock(address, timeout)
}

// functions that talk to the clock
func get_status(dial dialer, address string, timeout time.Duration) error {
	conn, err := dial(address, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = fmt.Fprint(conn, locator_commands["device_query"])
	if err != nil {
		return fmt.Errorf("sending status query to %s: %w", address, err)
	}
	fmt.Printf("sent status query to %s\n", address)

	udp_resp := make([]byte, maxBufferSize) // buffer for UDP responses
	packet_size, err := conn.Read(udp_resp)
	if err == nil {
		fmt.Println("response hexdump:")
		fmt.Printf("%s", hex.Dump(udp_resp[:packet_size]))

		if packet_size == 35 {
			// API version 1.x
			fmt.Printf("packet length %d (API version 1.x)\n", packet_size)
			struct_resp := Response10{}
			buf := bytes.NewReader(udp_resp)
			err = binary.Read(buf, binary.BigEndian, &struct_resp)
			if err != nil {
				return fmt.Errorf("decoding %d-byte response as API 1.x: %w", packet_size, err)
			}

			fmt.Printf("Type %x\n", struct_resp.DeviceType)
			fmt.Printf("IP %v\n", struct_resp.ClientIP)
			fmt.Printf("MAC %x\n", struct_resp.MAC_address)
			fmt.Printf("Ver %x\n", struct_resp.FirmwareVer)
			fmt.Printf("Syncs %d\n", struct_resp.NTPSyncCnt)
			fmt.Printf("Time %d\n", struct_resp.DisplayTime)
			fmt.Printf("Name %s\n", struct_resp.DeviceName)
		} else if packet_size == 40 {
			// API version 2.0
			fmt.Printf("packet length %d (API version 2.0)\n", packet_size)
			struct_resp := Response20{}
			buf := bytes.NewReader(udp_resp)
			err = binary.Read(buf, binary.BigEndian, &struct_resp)
			if err != nil {
				return fmt.Errorf("decoding %d-byte response as API 2.0: %w", packet_size, err)
			}

			fmt.Printf("Type %x\n", struct_resp.DeviceType)
			fmt.Printf("IP %v\n", struct_resp.ClientIP)
			fmt.Printf("MAC %x\n", struct_resp.MAC_address)
			fmt.Printf("Ver %x\n", struct_resp.FirmwareVer)
			fmt.Printf("Syncs %d\n", struct_resp.NTPSyncCnt)
			fmt.Printf("Time %d:%d:%d.%d\n", struct_resp.DisplayTime.Hour,
				struct_resp.DisplayTime.Minute,
				struct_resp.DisplayTime.Second,
				struct_resp.DisplayTime.Tenths)
			fmt.Printf("Mode %s\n", display_mode_string(struct_resp.DisplayMode))
			if struct_resp.DownAlarm&0x80 == 0x80 {
				fmt.Printf("Down alarm on (%d seconds)\n", struct_resp.DownAlarm&0x7f)
			} else {
				fmt.Println("Down alarm off")
			}
			fmt.Printf("Days %d\n", uint16(struct_resp.Days)<<3|uint16(struct_resp.Digits>>5))
			switch struct_resp.Digits & 0x1f {
			case 0:
				fmt.Println("Digits 4/6")
			case 1:
				fmt.Println("Digits (D):H:M:S")
			case 2:
				fmt.Println("Digits (H):M:S.Tenths")
			default:
				fmt.Printf("Digits unknown (%d)\n", struct_resp.Digits&0x1f)
			}
			if struct_resp.WifiSignal == 0 {
				fmt.Println("Wifi wired")
			} else {
				fmt.Printf("Wifi -%d dBm\n", struct_resp.WifiSignal)
			}
			fmt.Printf("Name %s\n", struct_resp.DeviceName)
		} else {
			fmt.Printf("packet length %d\n", packet_size)
			return fmt.Errorf("unexpected number of bytes returned so we don't know which protocol it is talking")
		}
	} else {
		return read_error(address, timeout, err)
	}
	return nil
}

func send_command(dial dialer, address string, timeout time.Duration, command string) error {
	conn, err := dial(address, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = fmt.Fprint(conn, locator_commands[command])
	if err != nil {
		return fmt.Errorf("sending command %s to %s: %w", command, address, err)
	}
	fmt.Printf("sent command %s to %s\n", command, address)

	udp_resp := make([]byte, maxBufferSize) // buffer for UDP responses
	packet_size, err := conn.Read(udp_resp)
	if err == nil {
		if packet_size != 2 {
			fmt.Printf("packet length %d\n", packet_size)
			return fmt.Errorf("unexpected packet size in UDP response")
		}
		if string(udp_resp[0]) != "A" {
			fmt.Println("response hexdump:")
			fmt.Printf("%s", hex.Dump(udp_resp[:packet_size]))
			return fmt.Errorf("response does not look like an acknowldgement")
		}
		fmt.Println("acked by clock")
	} else {
		return read_error(address, timeout, err)
	}
	return nil
}

func extract_time_part(time string, part int) (uint8, error) {
	time_components := strings.Split(time, ":")

	if len(time_components) > part {
		intVar, err := strconv.Atoi(time_components[part])
		if err != nil {
			return 0, fmt.Errorf("parsing %q as a time component: %w", time_components[part], err)
		}
		if intVar < 0 || intVar > 255 {
			return 0, fmt.Errorf("time component %q out of range (must be 0-255)", time_components[part])
		}
		return uint8(intVar), nil
	}
	return uint8(0), nil
}

func send_set_command(dial dialer, address string, timeout time.Duration, command string, time_string string) error {
	var err error
	set_struct := SetTimer{}
	set_struct.Command = uint8(locator_commands[command][0])

	set_struct.Hour, err = extract_time_part(time_string, 0)
	if err != nil {
		return err
	}
	set_struct.Minute, err = extract_time_part(time_string, 1)
	if err != nil {
		return err
	}
	set_struct.Second, err = extract_time_part(time_string, 2)
	if err != nil {
		return err
	}
	set_struct.Tenths, err = extract_time_part(time_string, 3)
	if err != nil {
		return err
	}
	set_struct.Hundredths, err = extract_time_part(time_string, 4)
	if err != nil {
		return err
	}

	fmt.Println(set_struct)

	conn, err := dial(address, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	var send_buf bytes.Buffer // buffer for UDP send
	err = binary.Write(&send_buf, binary.BigEndian, set_struct)
	if err != nil {
		return fmt.Errorf("encoding SetTimer struct: %w", err)
	}

	length, err := conn.Write(send_buf.Bytes())
	if err != nil {
		return fmt.Errorf("sending command %s to %s: %w", command, address, err)
	}
	fmt.Printf("sent command %s to %s (%d bytes)\n", command, address, length)

	udp_resp := make([]byte, maxBufferSize) // buffer for UDP responses
	packet_size, err := conn.Read(udp_resp)
	if err == nil {
		if packet_size != 2 {
			fmt.Printf("packet length %d\n", packet_size)
			return fmt.Errorf("unexpected packet size in UDP response")
		}
		if string(udp_resp[0]) != "A" {
			fmt.Println("response hexdump:")
			fmt.Printf("%s", hex.Dump(udp_resp[:packet_size]))
			return fmt.Errorf("response does not look like an acknowldgement")
		}
		fmt.Println("acked by clock")
	} else {
		return read_error(address, timeout, err)
	}
	return nil
}

// functions that build the command-line interface

// command_flagset builds the flag set for a subcommand, carrying the
// shared clock flags
func command_flagset(cfg *clockConfig, name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	cfg.register(fs)
	return fs
}

// require_address validates that a subcommand got exactly one
// positional argument - the clock address
func require_address(name string, args []string) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("%s requires exactly 1 argument (the clock address), got %d", name, len(args))
	}
	return args[0], nil
}

// status_command builds the subcommand that queries a clock's status
func status_command(cfg *clockConfig) *ffcli.Command {
	return &ffcli.Command{
		Name:       "status",
		ShortUsage: "ctm status [flags] <address>",
		ShortHelp:  "query the clock and print its status",
		FlagSet:    command_flagset(cfg, "status"),
		Exec: func(_ context.Context, args []string) error {
			address, err := require_address("status", args)
			if err != nil {
				return err
			}
			return get_status(cfg.dial_clock_or_test, cfg.addrport(address), cfg.timeout)
		},
	}
}

// mode_command builds a subcommand that sends one mode-switch command
// to the clock at the given address argument
func mode_command(cfg *clockConfig, name string, command string, help string) *ffcli.Command {
	return &ffcli.Command{
		Name:       name,
		ShortUsage: "ctm " + name + " [flags] <address>",
		ShortHelp:  help,
		FlagSet:    command_flagset(cfg, name),
		Exec: func(_ context.Context, args []string) error {
			address, err := require_address(name, args)
			if err != nil {
				return err
			}
			return send_command(cfg.dial_clock_or_test, cfg.addrport(address), cfg.timeout, command)
		},
	}
}

// set_time_command builds a subcommand that sends a SetTimer struct
// for the named timer command (up_set_time or down_set_time)
func set_time_command(cfg *clockConfig, name string, command string, help string) *ffcli.Command {
	return &ffcli.Command{
		Name:       name,
		ShortUsage: "ctm " + name + " [flags] <address> H:M:S:tenths:hundredths",
		ShortHelp:  help,
		FlagSet:    command_flagset(cfg, name),
		Exec: func(_ context.Context, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("%s requires exactly 2 arguments (clock address and H:M:S time), got %d", name, len(args))
			}
			return send_set_command(cfg.dial_clock_or_test, cfg.addrport(args[0]), cfg.timeout, command, args[1])
		},
	}
}

// new_command_tree builds the full ctm command tree around the given
// config.  main() and the tests share this so the dispatch logic is
// exercised exactly as shipped.
func new_command_tree(config *clockConfig) *ffcli.Command {
	root_flags := flag.NewFlagSet("ctm", flag.ExitOnError)
	config.register(root_flags)

	root := &ffcli.Command{
		Name:       "ctm",
		ShortUsage: "ctm [flags] <subcommand> [flags] <address> ...",
		ShortHelp:  "control Time Machines Corp. network clocks over UDP",
		FlagSet:    root_flags,
		Subcommands: []*ffcli.Command{
			status_command(config),
			mode_command(config, "time", "time_mode", "put the clock into time display mode"),
			mode_command(config, "up_ms", "up_mode_ms", "put the uptimer in minutes:seconds mode"),
			mode_command(config, "up_hms", "up_mode_hms", "put the uptimer in hours:minutes:seconds mode"),
			mode_command(config, "up_run", "up_mode_run", "run the uptimer"),
			mode_command(config, "up_pause", "up_mode_pause", "pause the uptimer"),
			mode_command(config, "up_reset_ms", "up_reset_ms", "reset the uptimer in minutes:seconds mode"),
			mode_command(config, "up_reset_hms", "up_reset_hms", "reset the uptimer in hours:minutes:seconds mode"),
			set_time_command(config, "up_set_time", "up_set_time", "set the uptimer to H:M:S:tenths:hundredths (smaller units optional)"),
			mode_command(config, "down_run", "down_mode_run", "run the downtimer"),
			mode_command(config, "down_pause", "down_mode_pause", "pause the downtimer"),
			set_time_command(config, "down_set_time", "down_set_time", "set the downtimer to H:M:S:tenths:hundredths (smaller units optional)"),
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

	root := new_command_tree(&config)

	err := root.ParseAndRun(context.Background(), os.Args[1:])
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(os.Stderr, "ctm: %v\n", err)
		}
		os.Exit(1)
	}
}
