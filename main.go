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
	Downtimer   uint8
	Unused      [2]uint8
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
// either before or after the subcommand name.
type clockConfig struct {
	timeout time.Duration
	port    int
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

// functions that talk to the clock
func get_status(address string, timeout time.Duration) error {
	conn, err := dial_clock(address, timeout)
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
		fmt.Printf("%s", hex.Dump(udp_resp))

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

			return fmt.Errorf("API 2.0 status decoding is not implemented yet")
		} else {
			fmt.Printf("packet length %d\n", packet_size)
			return fmt.Errorf("unexpected number of bytes returned so we don't know which protocol it is talking")
		}
	} else {
		return read_error(address, timeout, err)
	}
	return nil
}

func send_command(address string, timeout time.Duration, command string) error {
	conn, err := dial_clock(address, timeout)
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
			fmt.Printf("%s", hex.Dump(udp_resp))
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
		return uint8(intVar), nil
	}
	return uint8(0), nil
}

func send_set_command(address string, timeout time.Duration, command string, time_string string) error {
	var err error
	set_struct := SetTimer{}
	set_struct.Command = uint8(locator_commands["up_set_time"][0])

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

	conn, err := dial_clock(address, timeout)
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
			fmt.Printf("%s", hex.Dump(udp_resp))
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
			return get_status(cfg.addrport(address), cfg.timeout)
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
			return send_command(cfg.addrport(address), cfg.timeout, command)
		},
	}
}

// up_set_time_command builds the subcommand that sets the uptimer time
func up_set_time_command(cfg *clockConfig) *ffcli.Command {
	return &ffcli.Command{
		Name:       "up_set_time",
		ShortUsage: "ctm up_set_time [flags] <address> H:M:S:tenths:hundredths",
		ShortHelp:  "set the uptimer to H:M:S:tenths:hundredths (smaller units optional)",
		FlagSet:    command_flagset(cfg, "up_set_time"),
		Exec: func(_ context.Context, args []string) error {
			if len(args) != 2 {
				return fmt.Errorf("up_set_time requires exactly 2 arguments (clock address and H:M:S time), got %d", len(args))
			}
			return send_set_command(cfg.addrport(args[0]), cfg.timeout, "up_set_time", args[1])
		},
	}
}

func main() {
	var config clockConfig

	root_flags := flag.NewFlagSet("ctm", flag.ExitOnError)
	config.register(root_flags)

	root := &ffcli.Command{
		Name:       "ctm",
		ShortUsage: "ctm [flags] <subcommand> [flags] <address> ...",
		ShortHelp:  "control Time Machines Corp. network clocks over UDP",
		FlagSet:    root_flags,
		Subcommands: []*ffcli.Command{
			status_command(&config),
			mode_command(&config, "time", "time_mode", "put the clock into time display mode"),
			mode_command(&config, "up_ms", "up_mode_ms", "put the uptimer in minutes:seconds mode"),
			mode_command(&config, "up_hms", "up_mode_hms", "put the uptimer in hours:minutes:seconds mode"),
			mode_command(&config, "up_run", "up_mode_run", "run the uptimer"),
			mode_command(&config, "up_pause", "up_mode_pause", "pause the uptimer"),
			mode_command(&config, "up_reset_ms", "up_reset_ms", "reset the uptimer in minutes:seconds mode"),
			mode_command(&config, "up_reset_hms", "up_reset_hms", "reset the uptimer in hours:minutes:seconds mode"),
			up_set_time_command(&config),
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

	err := root.ParseAndRun(context.Background(), os.Args[1:])
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(os.Stderr, "ctm: %v\n", err)
		}
		os.Exit(1)
	}
}
