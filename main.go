package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
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
	Command		uint8
	Hour		uint8
	Minute		uint8
	Second		uint8
	Tenths		uint8
	Hundredths	uint8
}

const maxBufferSize = 48 // the biggest response packet is 40 bytes

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

// functions that talk to the clock
func get_status(address string) {
	conn, err := net.Dial("udp", address)
	defer conn.Close()
	if err != nil {
		fmt.Printf("Dial error %v\n", err)
		return
	}
	fmt.Fprintf(conn, locator_commands["device_query"])
	fmt.Printf("sent status query to %s\n", address)

	udp_resp := make([]byte, maxBufferSize) // buffer for UDP responses
	packet_size, err := bufio.NewReader(conn).Read(udp_resp)
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
				panic(err)
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

			fmt.Println("need a clock to test....")
			panic("unimplemented API 2.0")

			fmt.Printf("Type %x\n", struct_resp.DeviceType)
			fmt.Printf("IP %v\n", struct_resp.ClientIP)
			fmt.Printf("MAC %x\n", struct_resp.MAC_address)
			fmt.Printf("Ver %x\n", struct_resp.FirmwareVer)
			fmt.Printf("Syncs %d\n", struct_resp.NTPSyncCnt)
			fmt.Printf("Time %d\n", struct_resp.DisplayTime)
			fmt.Printf("Name %s\n", struct_resp.DeviceName)

			fmt.Printf("Mode %x\n", struct_resp.DisplayMode)
			if (struct_resp.DisplayMode & 0x40) == 0x40 {
				fmt.Println("\trunning")
			} else {
				fmt.Println("\tstopped")
			}
			if (struct_resp.DisplayMode & 0x80) == 0x80 {
				fmt.Println("\tdisplay M:S:Tenths")
			} else {
				fmt.Println("\tdisplay H:M:S")
			}
			fmt.Printf("Down %x\n", struct_resp.Downtimer)
			fmt.Printf("Wifi %x\n", struct_resp.WifiSignal)
		} else {
			fmt.Printf("packet length %d\n", packet_size)
			panic("unexpected number of bytes returned so we don't know which protocol it is talking")
		}
	} else {
		fmt.Printf("Some error %v\n", err)
	}
}

func send_command(address string, command string) {
	conn, err := net.Dial("udp", address)
	defer conn.Close()
	if err != nil {
		fmt.Printf("Dial error %v\n", err)
		return
	}
	fmt.Fprintf(conn, locator_commands[command])
	fmt.Printf("sent command %s to %s\n", command, address)

	udp_resp := make([]byte, maxBufferSize) // buffer for UDP responses
	packet_size, err := bufio.NewReader(conn).Read(udp_resp)
	if err == nil {
		if packet_size != 2 {
			fmt.Printf("packet length %d\n", packet_size)
			panic("unexpected packet size in UDP response")
		}
		if string(udp_resp[0]) != "A" {
			fmt.Println("response hexdump:")
			fmt.Printf("%s", hex.Dump(udp_resp))
			panic("response does not look like an acknowldgement")
		}
		fmt.Println("acked by clock")
	} else {
		fmt.Printf("Some error %v\n", err)
	}
}

func extract_time_part(time string, part int) uint8 {
	time_components := strings.Split(time, ":")
	// fmt.Println(time_components)

	if len(time_components) > part {
		intVar, err := strconv.Atoi(time_components[part])
		if err != nil {
			fmt.Printf("Atoi(%s) error %v\n", time_components[part], err)
			panic("Atoi failed.")
		}
		return uint8(intVar)
	} else {
		return uint8(0)
	}
}

func send_set_command(address string, command string, time string) {
	set_struct := SetTimer{}
	set_struct.Command = uint8(locator_commands["up_set_time"][0])

	set_struct.Hour = extract_time_part(time, 0)
	set_struct.Minute = extract_time_part(time, 1)
	set_struct.Second = extract_time_part(time, 2)
	set_struct.Tenths = extract_time_part(time, 3)
	set_struct.Hundredths = extract_time_part(time, 4)

	fmt.Println(set_struct)

	conn, err := net.Dial("udp", address)
	defer conn.Close()
	if err != nil {
		fmt.Printf("Dial error %v\n", err)
		return
	}

	var send_buf bytes.Buffer // buffer for UDP send
	err = binary.Write(&send_buf, binary.BigEndian, set_struct)
	if err != nil {
		panic(err)
	}

	length, err := conn.Write(send_buf.Bytes())
	if err != nil {
		panic(err)
	}
//	fmt.Fprintf(conn, send_buf)
	fmt.Printf("sent command %s to %s (%i bytes)\n", command, address, length)

	udp_resp := make([]byte, maxBufferSize) // buffer for UDP responses
	packet_size, err := bufio.NewReader(conn).Read(udp_resp)
	if err == nil {
		if packet_size != 2 {
			fmt.Printf("packet length %d\n", packet_size)
			panic("unexpected packet size in UDP response")
		}
		if string(udp_resp[0]) != "A" {
			fmt.Println("response hexdump:")
			fmt.Printf("%s", hex.Dump(udp_resp))
			panic("response does not look like an acknowldgement")
		}
		fmt.Println("acked by clock")
	} else {
		fmt.Printf("Some error %v\n", err)
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("expected arguments of subcommand and address")
		os.Exit(1)
	}

	clock_address := os.Args[2]
	clock_addrport := clock_address + ":7372"
	// fmt.Println(clock_addrport)

	switch os.Args[1] {
	case "status":
		get_status(clock_addrport)
	case "time":
		send_command(clock_addrport, "time_mode")
	case "up_ms":
		send_command(clock_addrport, "up_mode_ms")
	case "up_hms":
		send_command(clock_addrport, "up_mode_hms")
	case "up_run":
		send_command(clock_addrport, "up_mode_run")
	case "up_pause":
		send_command(clock_addrport, "up_mode_pause")
	case "up_reset_ms":
		send_command(clock_addrport, "up_reset_ms")
	case "up_reset_hms":
		send_command(clock_addrport, "up_reset_hms")
	case "up_set_time":
		set_time := os.Args[3]
		send_set_command(clock_addrport, "up_set_time", set_time) // but don't be upset :)
	default:
		panic("undefined subcommand")
	}
}
