package main

import (
    "bufio"
    "encoding/hex"
    "fmt"
    "net"
    "os"
)

const maxBufferSize = 64
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
    } else {
        fmt.Printf("Some error %v\n", err)
    }
    conn.Close()
}
