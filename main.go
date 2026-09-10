package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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

const udpTimeout = 2 * time.Second // how long to wait for a clock to respond

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

// convert a failed clock read into a clear error, calling out the
// timeout case so a silent clock is not mistaken for a crash
func read_error(address string, err error) error {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("clock at %s did not respond (timeout after %s)", address, udpTimeout)
	}
	return fmt.Errorf("reading response from %s: %w", address, err)
}

// dial the clock and arm a read deadline so a silent clock cannot
// block the caller forever
func dial_clock(address string) (net.Conn, error) {
	conn, err := net.Dial("udp", address)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", address, err)
	}
	err = conn.SetReadDeadline(time.Now().Add(udpTimeout))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("setting read deadline on %s: %w", address, err)
	}
	return conn, nil
}

// functions that talk to the clock
func get_status(address string) error {
	conn, err := dial_clock(address)
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
		return read_error(address, err)
	}
	return nil
}

func send_command(address string, command string) error {
	conn, err := dial_clock(address)
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
		return read_error(address, err)
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

func send_set_command(address string, command string, time string) error {
	var err error
	set_struct := SetTimer{}
	set_struct.Command = uint8(locator_commands["up_set_time"][0])

	set_struct.Hour, err = extract_time_part(time, 0)
	if err != nil {
		return err
	}
	set_struct.Minute, err = extract_time_part(time, 1)
	if err != nil {
		return err
	}
	set_struct.Second, err = extract_time_part(time, 2)
	if err != nil {
		return err
	}
	set_struct.Tenths, err = extract_time_part(time, 3)
	if err != nil {
		return err
	}
	set_struct.Hundredths, err = extract_time_part(time, 4)
	if err != nil {
		return err
	}

	fmt.Println(set_struct)

	conn, err := dial_clock(address)
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
		return read_error(address, err)
	}
	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("expected arguments of subcommand and address")
		os.Exit(1)
	}

	clock_address := os.Args[2]
	clock_addrport := clock_address + ":7372"

	err := func() error {
		switch os.Args[1] {
		case "status":
			return get_status(clock_addrport)
		case "time":
			return send_command(clock_addrport, "time_mode")
		case "up_ms":
			return send_command(clock_addrport, "up_mode_ms")
		case "up_hms":
			return send_command(clock_addrport, "up_mode_hms")
		case "up_run":
			return send_command(clock_addrport, "up_mode_run")
		case "up_pause":
			return send_command(clock_addrport, "up_mode_pause")
		case "up_reset_ms":
			return send_command(clock_addrport, "up_reset_ms")
		case "up_reset_hms":
			return send_command(clock_addrport, "up_reset_hms")
		case "up_set_time":
			set_time := os.Args[3]
			return send_set_command(clock_addrport, "up_set_time", set_time) // but don't be upset :)
		default:
			panic("undefined subcommand")
		}
	}()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ctm: %v\n", err)
		os.Exit(1)
	}
}
