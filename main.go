package main

import (
    "bufio"
    "bytes"
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "net"
    "os"
)

type IPAddr [4]byte

// API 1.x structures
type Time10 struct {
    Hour     uint8
    Minute   uint8
    Second   uint8
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
    Hour     uint8
    Minute   uint8
    Second   uint8
    Tenths   uint8
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

const maxBufferSize = 48 // the biggest response packet is 40 bytes
const device_query = "\xa1\x04\xb2"

func (ip IPAddr) String() string {
    return fmt.Sprintf("%v.%v.%v.%v", int(ip[0]), int(ip[1]), int(ip[2]), int(ip[3]))
}

func main() {
    udp_resp :=  make([]byte, maxBufferSize) // buffer for UDP responses

    clock_address := os.Args[1]
    clock_addrport := clock_address + ":7372"

    conn, err := net.Dial("udp", clock_addrport)
    if err != nil {
        fmt.Printf("Dial error %v\n", err)
        return
    }
    fmt.Println("connected")
    fmt.Fprintf(conn, device_query)
    fmt.Println("sent query")

    packet_size, err := bufio.NewReader(conn).Read(udp_resp)
    if err == nil {
	fmt.Println("read response")
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
	    // struct_resp := Response20{}
	    fmt.Println("need a clock to test....")
	    panic("unimplemented API 2.0")
//            fmt.Printf("Mode %x\n", struct_resp.DisplayMode)
//            if (struct_resp.DisplayMode & 0x40) == 0x40 {
//	        fmt.Println("\trunning")
//	    } else {
//	        fmt.Println("\tstopped")
//	    }
//            if (struct_resp.DisplayMode & 0x80) == 0x80 {
//	        fmt.Println("\tdisplay M:S:Tenths")
//	    } else {
//	        fmt.Println("\tdisplay H:M:S")
//	    }
//            fmt.Printf("Down %x\n", struct_resp.Downtimer)
//            fmt.Printf("Wifi %x\n", struct_resp.WifiSignal)
        } else {
	    fmt.Printf("packet length %d\n", packet_size)
	    panic("unexpected number of bytes returned so we don't know which protocol it is talking")
	}

	fmt.Println("code more....")
    } else {
        fmt.Printf("Some error %v\n", err)
    }
    conn.Close()
}
