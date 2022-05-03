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

type Time struct {
    Hour     uint8
    Minute   uint8
    Second   uint8
//    Tenths   uint8
}

type Response struct {
    DeviceType  uint8
    ClientIP    [4]uint8
    MAC_address [6]uint8
    FirmwareVer [2]uint8
    NTPSyncCnt  uint16
    DisplayTime Time
//    DisplayMode uint8
//    Downtimer   uint8
//    Unused      [2]uint8
//    WifiSignal  uint8
    DeviceName  [16]uint8
}

const maxBufferSize = 64 // the biggest response packet is 40 bytes
const device_query = "\xa1\x04\xb2"

func main() {
    udp_resp :=  make([]byte, maxBufferSize) // buffer for UDP responses

    clock_address := os.Args[1]
    clock_addrport := clock_address + ":7372"

    conn, err := net.Dial("udp", clock_addrport)
    if err != nil {
        fmt.Printf("Some error %v", err)
        return
    }
    fmt.Println("connected")
    fmt.Fprintf(conn, device_query)
    fmt.Println("sent query")
    _, err = bufio.NewReader(conn).Read(udp_resp)
    if err == nil {
	fmt.Println("read response")
	fmt.Printf("%s", hex.Dump(udp_resp))

	struct_resp := Response{}
	buf := bytes.NewReader(udp_resp)
	err = binary.Read(buf, binary.BigEndian, &struct_resp)
	if err != nil {
            panic(err)
        }

        fmt.Printf("Type %x\n", struct_resp.DeviceType)
        fmt.Printf("IP %x\n", struct_resp.ClientIP)
        fmt.Printf("MAC %x\n", struct_resp.MAC_address)
        fmt.Printf("Ver %x\n", struct_resp.FirmwareVer)
        fmt.Printf("Syncs %d\n", struct_resp.NTPSyncCnt)
        fmt.Printf("Time %d\n", struct_resp.DisplayTime)
//        fmt.Printf("Mode %x\n", struct_resp.DisplayMode)
//        if (struct_resp.DisplayMode & 0x40) == 0x40 {
//	    fmt.Println("\trunning")
//	} else {
//	    fmt.Println("\tstopped")
//	}
//        if (struct_resp.DisplayMode & 0x80) == 0x80 {
//	    fmt.Println("\tdisplay M:S:Tenths")
//	} else {
//	    fmt.Println("\tdisplay H:M:S")
//	}
//        fmt.Printf("Down %x\n", struct_resp.Downtimer)
//        fmt.Printf("Wifi %x\n", struct_resp.WifiSignal)
        fmt.Printf("Name %s\n", struct_resp.DeviceName)

	fmt.Println("code more....")
    } else {
        fmt.Printf("Some error %v\n", err)
    }
    conn.Close()
}
