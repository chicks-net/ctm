package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeConn stands in for a clock connection: it records everything the
// program writes and replays canned reads.  If the program reads past
// the canned data the fake reports the packet-size mismatch exactly
// like a real socket would not - the functions under test never
// should, and the dispatch tests below assert they do not.
type fakeConn struct {
	sendBuf     bytes.Buffer // what the program writes
	reply       []byte       // canned packet for the first Read
	closeCalled bool
}

func (fc *fakeConn) Write(p []byte) (int, error) {
	return fc.sendBuf.Write(p)
}

func (fc *fakeConn) Read(p []byte) (int, error) {
	n := copy(p, fc.reply)
	if n == 0 {
		return 0, fmt.Errorf("fake clock has no more data")
	}
	fc.reply = nil
	return n, nil
}

func (fc *fakeConn) Close() error {
	fc.closeCalled = true
	return nil
}

func (fc *fakeConn) LocalAddr() net.Addr                { return nil }
func (fc *fakeConn) RemoteAddr() net.Addr               { return nil }
func (fc *fakeConn) SetDeadline(t time.Time) error      { return nil }
func (fc *fakeConn) SetReadDeadline(t time.Time) error  { return nil }
func (fc *fakeConn) SetWriteDeadline(t time.Time) error { return nil }

// newFakeConfig builds a clockConfig whose dialer hands out the given
// canned reply, and returns the config plus a channel receiving each
// connection the program opened.
func newFakeConfig(reply []byte) (*clockConfig, chan *fakeConn) {
	conns := make(chan *fakeConn, 4)
	cfg := &clockConfig{
		timeout: time.Second,
		port:    defaultPort,
		dial: func(address string, timeout time.Duration) (net.Conn, error) {
			fc := &fakeConn{reply: reply}
			conns <- fc
			return fc, nil
		},
	}
	return cfg, conns
}

// runCommand drives the real command tree with argv, mirroring what
// main() does with os.Args
func runCommand(t *testing.T, cfg *clockConfig, argv ...string) error {
	t.Helper()
	root := newCommandTree(cfg)
	return root.ParseAndRun(context.Background(), argv)
}

// TestSetTimerWireFormat pins the 6-byte big-endian payload that
// up_set_time/down_set_time put on the wire
func TestSetTimerWireFormat(t *testing.T) {
	set := SetTimer{Command: 0xaa, Hour: 1, Minute: 2, Second: 3, Tenths: 4, Hundredths: 5}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, set); err != nil {
		t.Fatalf("encoding SetTimer: %v", err)
	}
	want := []byte{0xaa, 1, 2, 3, 4, 5}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("SetTimer encoded to % x, want % x", buf.Bytes(), want)
	}
}

// TestSetColorWireFormat pins the 7-byte big-endian Color Set packet
// from API 2.0 section 1.4.6: 0xB6, then the MM:SS digit color,
// then the HH digit color
func TestSetColorWireFormat(t *testing.T) {
	color := SetColor{
		Command: 0xb6,
		MMSS:    [3]uint8{0xff, 0x00, 0x00},
		HH:      [3]uint8{0x00, 0xff, 0x00},
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, color); err != nil {
		t.Fatalf("encoding SetColor: %v", err)
	}
	want := []byte{0xb6, 0xff, 0x00, 0x00, 0x00, 0xff, 0x00}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("SetColor encoded to % x, want % x", buf.Bytes(), want)
	}
	var back SetColor
	if err := binary.Read(&buf, binary.BigEndian, &back); err != nil {
		t.Fatalf("decoding SetColor: %v", err)
	}
	if back != color {
		t.Errorf("SetColor round-trip mismatch: got %+v, want %+v", back, color)
	}
}

// TestParseColorSpec covers the rrggbb / rrggbb:rrggbb color syntax:
// one color applies to all digits, two set MM:SS and HH independently
func TestParseColorSpec(t *testing.T) {
	red := [3]uint8{0xff, 0x00, 0x00}
	green := [3]uint8{0x00, 0xff, 0x00}
	tests := []struct {
		spec  string
		mmss  [3]uint8
		hh    [3]uint8
		fails bool
	}{
		{spec: "ff0000", mmss: red, hh: red},                                               // all digits red
		{spec: "FFAA00", mmss: [3]uint8{0xff, 0xaa, 0x00}, hh: [3]uint8{0xff, 0xaa, 0x00}}, // uppercase hex accepted
		{spec: "ff0000:00ff00", mmss: red, hh: green},                                      // MM:SS red, HH green
		{spec: "00ff00:ff0000", mmss: green, hh: red},
		{spec: "ff00", fails: true},                 // too short
		{spec: "ff00000", fails: true},              // too long
		{spec: "gg0000", fails: true},               // not hex
		{spec: "ff0000:00ff00:0000ff", fails: true}, // more than two colors
		{spec: "ff0000:0", fails: true},             // second color malformed
		{spec: "", fails: true},                     // empty spec
	}
	for _, tc := range tests {
		mmss, hh, err := parseColorSpec(tc.spec)
		if tc.fails {
			if err == nil {
				t.Errorf("parseColorSpec(%q) = %+v/%+v, want error", tc.spec, mmss, hh)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseColorSpec(%q) unexpected error: %v", tc.spec, err)
			continue
		}
		if mmss != tc.mmss {
			t.Errorf("parseColorSpec(%q) mmss = %+v, want %+v", tc.spec, mmss, tc.mmss)
		}
		if hh != tc.hh {
			t.Errorf("parseColorSpec(%q) hh = %+v, want %+v", tc.spec, hh, tc.hh)
		}
	}
}

// TestResponse20WireFormat pins the 40-byte API 2.0 status packet
func TestResponse20WireFormat(t *testing.T) {
	r20 := Response20{
		DeviceType:   0x01,
		ClientIP:     IPAddr{192, 168, 42, 204},
		MACAddress:   [6]uint8{0x70, 0xb3, 0xd5, 0x75, 0x68, 0xe2},
		FirmwareVer:  [2]uint8{5, 0},
		NTPSyncCount: 4654,
		DisplayTime:  Time20{Hour: 12, Minute: 34, Second: 56, Tenths: 7},
		DisplayMode:  0x01 | 0x40,
		DownAlarm:    0x85,
		Days:         3,
		Digits:       2,
		WifiSignal:   58,
		DeviceName:   [16]uint8{'P', 'O', 'E', '_', 'C', 'l', 'o', 'c', 'k'},
	}
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, r20); err != nil {
		t.Fatalf("encoding Response20: %v", err)
	}
	if buf.Len() != 40 {
		t.Errorf("Response20 encoded to %d bytes, want 40", buf.Len())
	}
	var back Response20
	if err := binary.Read(&buf, binary.BigEndian, &back); err != nil {
		t.Fatalf("decoding Response20: %v", err)
	}
	if back != r20 {
		t.Errorf("Response20 round-trip mismatch: got %+v, want %+v", back, r20)
	}
}

// TestResponse10WireFormat pins the API 1.x status packet: the struct
// sums to 34 bytes and the protocol adds one trailing padding byte,
// so the packet on the wire is 35
func TestResponse10WireFormat(t *testing.T) {
	var r10 Response10
	if err := binary.Read(bytes.NewReader(r10Bytes()), binary.BigEndian, &r10); err != nil {
		t.Fatalf("decoding 35-byte response as API 1.x: %v", err)
	}
	if r10.DeviceType != 0x01 {
		t.Errorf("DeviceType = %#x, want 0x01", r10.DeviceType)
	}
	if r10.ClientIP.String() != "192.168.42.204" {
		t.Errorf("ClientIP = %s, want 192.168.42.204", r10.ClientIP)
	}
	if got := string(bytes.TrimRight(r10.DeviceName[:], "\x00")); got != "POE_Clock_UTC" {
		t.Errorf("DeviceName = %q, want POE_Clock_UTC", got)
	}
}

// r10Bytes returns a 35-byte API 1.x packet matching the README's
// hexdump, with one trailing padding byte the struct does not decode
func r10Bytes() []byte {
	b := []byte{
		0x01,              // DeviceType
		192, 168, 42, 204, // ClientIP
		0x70, 0xb3, 0xd5, 0x75, 0x68, 0xe2, // MAC
		0x05, 0x00, // FirmwareVer
		0x12, 0x2e, // NTPSyncCount
		2, 42, 44, // DisplayTime
	}
	b = append(b, "POE_Clock_UTC"...)
	for len(b) < 34 {
		b = append(b, 0)
	}
	b = append(b, 0) // 35th byte: ignored trailing padding
	return b
}

// TestTimeStructSizes pins the wire sizes of the time sub-structs
func TestTimeStructSizes(t *testing.T) {
	if sz := binary.Size(Time10{}); sz != 3 {
		t.Errorf("Time10 wire size = %d, want 3", sz)
	}
	if sz := binary.Size(Time20{}); sz != 4 {
		t.Errorf("Time20 wire size = %d, want 4", sz)
	}
	if sz := binary.Size(SetTimer{}); sz != 6 {
		t.Errorf("SetTimer wire size = %d, want 6", sz)
	}
	if sz := binary.Size(SetColor{}); sz != 7 {
		t.Errorf("SetColor wire size = %d, want 7", sz)
	}
	if sz := binary.Size(Response10{}); sz != 34 {
		t.Errorf("Response10 wire size = %d, want 34 (packet adds one pad byte)", sz)
	}
	if sz := binary.Size(Response20{}); sz != 40 {
		t.Errorf("Response20 wire size = %d, want 40", sz)
	}
}

// TestLocatorCommands pins the command bytes against typos and the
// two historical #19 bugs (up_set_time/down_set_time opcodes)
func TestLocatorCommands(t *testing.T) {
	want := map[string]string{
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
		"color_set":       "\xb6",
	}
	if len(locatorCommands) != len(want) {
		t.Errorf("locatorCommands has %d entries, want %d", len(locatorCommands), len(want))
	}
	for name, wantBytes := range want {
		if got := locatorCommands[name]; got != wantBytes {
			t.Errorf("locatorCommands[%q] = % x, want % x", name, got, wantBytes)
		}
	}
}

// TestExtractTimePart covers parsing of the H:M:S:tenths:hundredths
// time syntax, including omitted trailing components and out-of-range
// rejection
func TestExtractTimePart(t *testing.T) {
	tests := []struct {
		time  string
		part  int
		want  uint8
		fails bool
	}{
		{time: "1:2:3:4:5", part: 0, want: 1},
		{time: "1:2:3:4:5", part: 4, want: 5},
		{time: "0:30", part: 0, want: 0},
		{time: "0:30", part: 1, want: 30},
		{time: "0:30", part: 2, want: 0}, // trailing components omitted -> 0
		{time: "0:30", part: 4, want: 0},
		{time: "1:2:x", part: 2, fails: true},   // non-numeric
		{time: "1:2:300", part: 2, fails: true}, // silent uint8 truncation guard
		{time: "1:2:-1", part: 2, fails: true},
		{time: "255:0:0", part: 0, want: 255},   // boundary: largest storable value
		{time: "256:0:0", part: 0, fails: true}, // one past the boundary
	}
	for _, tc := range tests {
		got, err := extractTimePart(tc.time, tc.part)
		if tc.fails {
			if err == nil {
				t.Errorf("extractTimePart(%q, %d) = %d, want error", tc.time, tc.part, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("extractTimePart(%q, %d) unexpected error: %v", tc.time, tc.part, err)
			continue
		}
		if got != tc.want {
			t.Errorf("extractTimePart(%q, %d) = %d, want %d", tc.time, tc.part, got, tc.want)
		}
	}
}

// TestDisplayModeString covers the API 2.0 display-mode decoding
func TestDisplayModeString(t *testing.T) {
	tests := []struct {
		mode uint8
		want string
	}{
		{mode: 0x00, want: "time"},
		{mode: 0x01, want: "up timer, stopped, H:M:S"},
		{mode: 0x41, want: "up timer, running, H:M:S"},
		{mode: 0x02, want: "down timer, stopped, H:M:S"},
		{mode: 0x03, want: "interval count up, stopped, H:M:S"},
		{mode: 0x04, want: "interval count down, stopped, H:M:S"},
		{mode: 0x07, want: "unknown (0x07), stopped, H:M:S"},
		{mode: 0xC1, want: "up timer, running, M:S.Tenths"},
		{mode: 0x61, want: "up timer, running, D:H:M"},
	}
	for _, tc := range tests {
		if got := displayModeString(tc.mode); got != tc.want {
			t.Errorf("displayModeString(%#x) = %q, want %q", tc.mode, got, tc.want)
		}
	}
}

// TestAddrport checks host:port joining
func TestAddrport(t *testing.T) {
	cfg := clockConfig{port: 7372}
	if got := cfg.addrport("192.168.42.204"); got != "192.168.42.204:7372" {
		t.Errorf("addrport = %q, want 192.168.42.204:7372", got)
	}
	cfg.port = 1234
	if got := cfg.addrport("clock.local"); got != "clock.local:1234" {
		t.Errorf("addrport = %q, want clock.local:1234", got)
	}
}

// TestIPAddrString checks dotted-quad formatting without leading-zero
// weirdness
func TestIPAddrString(t *testing.T) {
	tests := []struct {
		ip   IPAddr
		want string
	}{
		{ip: IPAddr{192, 168, 42, 204}, want: "192.168.42.204"},
		{ip: IPAddr{0, 0, 0, 0}, want: "0.0.0.0"},
		{ip: IPAddr{10, 1, 2, 3}, want: "10.1.2.3"},
	}
	for _, tc := range tests {
		if got := tc.ip.String(); got != tc.want {
			t.Errorf("IPAddr(%v).String() = %q, want %q", tc.ip, got, tc.want)
		}
	}
}

// TestSendCommandDispatch drives the real ffcli tree for every mode
// subcommand and checks the exact bytes each one puts on the wire
func TestSendCommandDispatch(t *testing.T) {
	// CLI verb -> internal command name
	verbFor := map[string]string{
		"time":         "time_mode",
		"up_ms":        "up_mode_ms",
		"up_hms":       "up_mode_hms",
		"up_pause":     "up_mode_pause",
		"up_run":       "up_mode_run",
		"up_reset_ms":  "up_reset_ms",
		"up_reset_hms": "up_reset_hms",
		"down_run":     "down_mode_run",
		"down_pause":   "down_mode_pause",
	}
	ack := []byte{'A', 0x00}
	for verb, name := range verbFor {
		wire := locatorCommands[name]
		t.Run(verb, func(t *testing.T) {
			cfg, conns := newFakeConfig(ack)
			err := runCommand(t, cfg, verb, "192.168.42.204")
			if err != nil {
				t.Fatalf("ctm %s: %v", verb, err)
			}
			select {
			case fc := <-conns:
				if got := fc.sendBuf.String(); got != wire {
					t.Errorf("%s sent % x, want % x", verb, got, wire)
				}
				if !fc.closeCalled {
					t.Errorf("%s did not close the connection", verb)
				}
			default:
				t.Fatal("no connection was opened")
			}
		})
	}
}

// TestSendCommandBadResponse makes sure a non-ack reply is an error
func TestSendCommandBadResponse(t *testing.T) {
	cfg, _ := newFakeConfig([]byte{'N', 0x00})
	err := runCommand(t, cfg, "up_run", "192.168.42.204")
	if err == nil {
		t.Fatal("expected an error when the clock does not ack")
	}
	if !strings.Contains(err.Error(), "acknowledgement") {
		t.Errorf("error %q does not mention the missing acknowledgement", err)
	}
}

// TestSendCommandWrongSize makes sure an unexpected packet size is an error
func TestSendCommandWrongSize(t *testing.T) {
	cfg, _ := newFakeConfig([]byte{'A'})
	err := runCommand(t, cfg, "up_run", "192.168.42.204")
	if err == nil {
		t.Fatal("expected an error for a short ack packet")
	}
	if !strings.Contains(err.Error(), "unexpected packet size") {
		t.Errorf("error %q does not mention packet size", err)
	}
}

// TestRequireAddress covers the arg-count validation for subcommands
func TestRequireAddress(t *testing.T) {
	if _, err := requireAddress("status", nil); err == nil {
		t.Error("expected an error with no args")
	}
	if _, err := requireAddress("status", []string{"a", "b"}); err == nil {
		t.Error("expected an error with two args")
	}
	addr, err := requireAddress("status", []string{"192.168.42.204"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addr != "192.168.42.204" {
		t.Errorf("address = %q, want 192.168.42.204", addr)
	}
}

// TestRequireAddressDispatch drives arg-count validation through the
// real command tree
func TestRequireAddressDispatch(t *testing.T) {
	tests := []struct {
		argv  []string
		fails bool
	}{
		{argv: []string{"time"}, fails: true},                       // missing address
		{argv: []string{"time", "a", "b"}, fails: true},             // extra args
		{argv: []string{"up_set_time", "a"}, fails: true},           // missing time
		{argv: []string{"up_set_time", "a", "b", "c"}, fails: true}, // extra args
		{argv: []string{"up_set_time", "192.168.42.204", "0:30"}},
	}
	for _, tc := range tests {
		cfg, conns := newFakeConfig([]byte{'A', 0x00})
		err := runCommand(t, cfg, tc.argv...)
		if tc.fails && err == nil {
			t.Errorf("ctm %v: expected an error", tc.argv)
		}
		if !tc.fails && err != nil {
			t.Errorf("ctm %v: %v", tc.argv, err)
		}
		if tc.fails && len(conns) > 0 {
			t.Errorf("ctm %v: should not have dialed", tc.argv)
		}
	}
}

// TestStatusDispatch checks the status subcommand against canned API
// 1.x and 2.0 packets
func TestStatusDispatch(t *testing.T) {
	t.Run("v1", func(t *testing.T) {
		cfg, conns := newFakeConfig(r10Bytes())
		err := runCommand(t, cfg, "status", "192.168.42.204")
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		select {
		case fc := <-conns:
			if got := fc.sendBuf.String(); got != locatorCommands["device_query"] {
				t.Errorf("status sent % x, want % x", got, locatorCommands["device_query"])
			}
		default:
			t.Fatal("no connection was opened")
		}
	})
	t.Run("v2", func(t *testing.T) {
		packet := make([]byte, 40)
		packet[0] = 0x01
		copy(packet[1:5], []byte{192, 168, 42, 204})
		copy(packet[5:11], []byte{0x70, 0xb3, 0xd5, 0x75, 0x68, 0xe2})
		copy(packet[11:13], []byte{0x05, 0x00})
		binary.BigEndian.PutUint16(packet[13:15], 4654)
		copy(packet[15:19], []byte{12, 34, 56, 7})
		packet[19] = 0x41 // up timer, running, H:M:S
		packet[20] = 0x85 // down alarm on, 5 seconds
		packet[21] = 3    // Days
		packet[22] = 0x22 // Digits: D:H:M:S
		packet[23] = 58   // Wifi -58 dBm
		copy(packet[24:40], "POE_Clock")

		cfg, _ := newFakeConfig(packet)
		if err := runCommand(t, cfg, "status", "192.168.42.204"); err != nil {
			t.Fatalf("status: %v", err)
		}
	})
	t.Run("unexpected size", func(t *testing.T) {
		cfg, _ := newFakeConfig(make([]byte, 37))
		err := runCommand(t, cfg, "status", "192.168.42.204")
		if err == nil {
			t.Fatal("expected an error for a 37-byte packet")
		}
		if !strings.Contains(err.Error(), "unexpected number of bytes") {
			t.Errorf("error %q does not mention the packet size problem", err)
		}
	})
}

// TestSetTimeDispatch checks that up_set_time/down_set_time send the
// expected 6-byte payloads, including the correct leading opcode for
// each (the historical #19 bug hardcoded the up opcode)
func TestSetTimeDispatch(t *testing.T) {
	tests := []struct {
		name string
		time string
		want []byte
	}{
		{name: "up_set_time", time: "1:2:3:4:5", want: []byte{0xaa, 1, 2, 3, 4, 5}},
		{name: "down_set_time", time: "10:0:0", want: []byte{0xab, 10, 0, 0, 0, 0}},
		{name: "up_set_time", time: "0:30", want: []byte{0xaa, 0, 30, 0, 0, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name+" "+tc.time, func(t *testing.T) {
			cfg, conns := newFakeConfig([]byte{'A', 0x00})
			err := runCommand(t, cfg, tc.name, "192.168.42.204", tc.time)
			if err != nil {
				t.Fatalf("ctm %s %s: %v", tc.name, tc.time, err)
			}
			select {
			case fc := <-conns:
				if got := fc.sendBuf.Bytes(); !bytes.Equal(got, tc.want) {
					t.Errorf("%s %s sent % x, want % x", tc.name, tc.time, got, tc.want)
				}
			default:
				t.Fatal("no connection was opened")
			}
		})
	}
}

// TestSetTimeRangeRejectDispatch drives the new 0-255 range check
// through the command tree: a bad component must fail before the
// clock is ever dialed
func TestSetTimeRangeRejectDispatch(t *testing.T) {
	cfg, conns := newFakeConfig([]byte{'A', 0x00})
	err := runCommand(t, cfg, "up_set_time", "192.168.42.204", "300:0:0")
	if err == nil {
		t.Fatal("expected out-of-range hours to be an error")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error %q does not mention the range problem", err)
	}
	if len(conns) > 0 {
		t.Error("should not have dialed the clock with an invalid time")
	}
}

// TestColorSetDispatch checks that color_set sends the exact 7-byte
// packets from API 2.0 section 1.4.6 for both the single-color and
// two-color forms
func TestColorSetDispatch(t *testing.T) {
	tests := []struct {
		spec string
		want []byte
	}{
		{spec: "ff0000", want: []byte{0xb6, 0xff, 0x00, 0x00, 0xff, 0x00, 0x00}},
		{spec: "ff0000:00ff00", want: []byte{0xb6, 0xff, 0x00, 0x00, 0x00, 0xff, 0x00}},
	}
	for _, tc := range tests {
		t.Run(tc.spec, func(t *testing.T) {
			cfg, conns := newFakeConfig([]byte{'A', 0x00})
			err := runCommand(t, cfg, "color_set", "192.168.42.204", tc.spec)
			if err != nil {
				t.Fatalf("ctm color_set %s: %v", tc.spec, err)
			}
			select {
			case fc := <-conns:
				if got := fc.sendBuf.Bytes(); !bytes.Equal(got, tc.want) {
					t.Errorf("color_set %s sent % x, want % x", tc.spec, got, tc.want)
				}
				if !fc.closeCalled {
					t.Error("color_set did not close the connection")
				}
			default:
				t.Fatal("no connection was opened")
			}
		})
	}
}

// TestColorSetRejectDispatch drives invalid color specs and bad arg
// counts through the command tree: they must fail before the clock
// is ever dialed
func TestColorSetRejectDispatch(t *testing.T) {
	tests := []struct {
		argv []string
	}{
		{argv: []string{"color_set"}},                                           // no args
		{argv: []string{"color_set", "192.168.42.204"}},                         // missing color
		{argv: []string{"color_set", "192.168.42.204", "ff0000", "extra"}},      // extra args
		{argv: []string{"color_set", "192.168.42.204", "ff00"}},                 // wrong length
		{argv: []string{"color_set", "192.168.42.204", "gg0000"}},               // not hex
		{argv: []string{"color_set", "192.168.42.204", "ff0000:00ff00:0000ff"}}, // 3 colors
	}
	for _, tc := range tests {
		t.Run(strings.Join(tc.argv, " "), func(t *testing.T) {
			cfg, conns := newFakeConfig([]byte{'A', 0x00})
			err := runCommand(t, cfg, tc.argv...)
			if err == nil {
				t.Errorf("ctm %v: expected an error", tc.argv)
			}
			if len(conns) > 0 {
				t.Errorf("ctm %v: should not have dialed", tc.argv)
			}
		})
	}
}

// TestFlagsInBothPositions checks that -port and -timeout are
// accepted before and after the subcommand name and actually reach
// the dialer
func TestFlagsInBothPositions(t *testing.T) {
	for _, argv := range [][]string{
		{"-port", "9999", "up_run", "192.168.42.204"},
		{"up_run", "-port", "9999", "192.168.42.204"},
	} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			var dialed string
			cfg := &clockConfig{
				dial: func(address string, timeout time.Duration) (net.Conn, error) {
					dialed = address
					return &fakeConn{reply: []byte{'A', 0x00}}, nil
				},
			}
			err := runCommand(t, cfg, argv...)
			if err != nil {
				t.Fatalf("ctm %v: %v", argv, err)
			}
			if dialed != "192.168.42.204:9999" {
				t.Errorf("dialed %q, want 192.168.42.204:9999", dialed)
			}
		})
	}
}

// TestFlagsTimeoutReachesDialer makes sure -timeout is passed to the
// dialer in either position
func TestFlagsTimeoutReachesDialer(t *testing.T) {
	for _, argv := range [][]string{
		{"-timeout", "5s", "up_run", "192.168.42.204"},
		{"up_run", "-timeout", "5s", "192.168.42.204"},
	} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			var got time.Duration
			cfg := &clockConfig{
				dial: func(address string, timeout time.Duration) (net.Conn, error) {
					got = timeout
					return &fakeConn{reply: []byte{'A', 0x00}}, nil
				},
			}
			err := runCommand(t, cfg, argv...)
			if err != nil {
				t.Fatalf("ctm %v: %v", argv, err)
			}
			if got != 5*time.Second {
				t.Errorf("dialer got timeout %v, want 5s", got)
			}
		})
	}
}

// TestHelpCommand checks that the built-in help subcommand runs
// cleanly.  (-h cannot be tested here: the flag sets use
// flag.ExitOnError, so -h calls os.Exit(0) and would kill the test
// binary - that behavior is only exercised by hand.)
func TestHelpCommand(t *testing.T) {
	cfg := &clockConfig{}
	if err := runCommand(t, cfg, "help"); err != nil {
		t.Errorf("ctm help: %v", err)
	}
}

// TestUnknownSubcommand checks the root Exec's fallback error
func TestUnknownSubcommand(t *testing.T) {
	cfg := &clockConfig{}
	err := runCommand(t, cfg, "frobnicate", "192.168.42.204")
	if err == nil {
		t.Fatal("expected an error for an unknown subcommand")
	}
	if !strings.Contains(err.Error(), "unknown subcommand") {
		t.Errorf("error %q does not mention the unknown subcommand", err)
	}
}

// TestDialClockHappyPath exercises the real dialer against a
// loopback UDP socket: written commands must arrive, replies must
// round-trip, and the deadline must not fire
func TestDialClockHappyPath(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Skipf("cannot listen on loopback UDP: %v", err)
	}
	defer server.Close()

	go func() {
		buf := make([]byte, maxBufferSize)
		n, client, err := server.ReadFrom(buf)
		if err != nil {
			return
		}
		_, _ = server.WriteTo(buf[:n], client)
	}()

	conn, err := dialClock(server.LocalAddr().String(), time.Second)
	if err != nil {
		t.Fatalf("dialClock: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, maxBufferSize)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf[:n]) != "ping" {
		t.Errorf("echoed %q, want ping", buf[:n])
	}
}

// TestDialClockTimeout arms the read deadline against a black-hole
// server and expects the friendly timeout error within the budget
func TestDialClockTimeout(t *testing.T) {
	blackhole, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Skipf("cannot listen on loopback UDP: %v", err)
	}
	defer blackhole.Close()

	const timeout = 300 * time.Millisecond
	start := time.Now()
	conn, err := dialClock(blackhole.LocalAddr().String(), timeout)
	if err != nil {
		t.Fatalf("dialClock: %v", err)
	}
	defer conn.Close()

	// ensure the UDP socket is connected; a failed write would hang
	// the read below until the deadline, so fail fast instead
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, maxBufferSize)
	_, readErr := conn.Read(buf)
	elapsed := time.Since(start)
	if readErr == nil {
		t.Fatal("expected a read error against the black-hole server")
	}
	if elapsed > 2*timeout {
		t.Errorf("read took %v, want it to give up around %v", elapsed, timeout)
	}
	wrapped := readError(blackhole.LocalAddr().String(), timeout, readErr)
	if !strings.Contains(wrapped.Error(), "did not respond") {
		t.Errorf("readError %q does not explain the timeout", wrapped)
	}
}

// TestDialClockBadAddress makes sure an undialable address is a
// clean error, not a panic
func TestDialClockBadAddress(t *testing.T) {
	_, err := dialClock("", time.Second)
	if err == nil {
		t.Fatal("expected an error for an empty address")
	}
	if !strings.Contains(err.Error(), "connecting to") {
		t.Errorf("error %q does not mention the connection failure", err)
	}
}

// --- fuzz targets for the untrusted-input parsers ---
//
// These run their seed corpora as ordinary tests under `go test ./...`,
// which is how the CI exercises them.  To actually fuzz (random inputs,
// minimizing engine, corpus growth in testdata/fuzz/), run e.g.
//
//	go test -fuzz FuzzExtractTimePart -fuzztime 30s
//
// The targets exist partly for real hardening - every one of these
// functions parses bytes that originate off the wire or off argv - and
// partly so OpenSSF Scorecard's Fuzzing check detects the repo as
// fuzzed (it looks for `func FuzzXxx(*testing.F)` in *_test.go files).

// FuzzExtractTimePart feeds arbitrary strings and part indexes to the
// time parser: no input may panic, and any parsed value must stay in
// the 0-255 range a uint8 component can hold.  Negative part indexes
// used to panic here - the guard in extractTimePart was added after
// this target found it.
func FuzzExtractTimePart(f *testing.F) {
	seeds := []struct {
		value string
		part  int
	}{
		{"1:2:3:4:5", 0},
		{"1:2:3:4:5", 4},
		{"0:30", 1},
		{"0:30", 2},
		{"255:0:0", 0},
		{"256:0:0", 0},
		{"1:2:-1", 2},
		{"1:2:x", 2},
		{"", 0},
		{"1:2:3:4:5", -1},
		{"1:2:3:4:5", 99},
	}
	for _, s := range seeds {
		f.Add(s.value, s.part)
	}
	f.Fuzz(func(t *testing.T, value string, part int) {
		got, err := extractTimePart(value, part)
		if err != nil {
			// erroring is fine; a nonzero value alongside an
			// error would mislead the caller
			if got != 0 {
				t.Fatalf("extractTimePart(%q, %d) = %d with error %v", value, part, got, err)
			}
			return
		}
		if got > 255 {
			t.Fatalf("extractTimePart(%q, %d) = %d, want 0-255", value, part, got)
		}
	})
}

// FuzzParseColorSpec feeds arbitrary color specs to the parser: no
// input may panic, and a successful parse must yield exactly the
// MM:SS and HH triples the syntax describes
func FuzzParseColorSpec(f *testing.F) {
	for _, seed := range []string{
		"ff0000",
		"FFAA00",
		"ff0000:00ff00",
		"00ff00:ff0000",
		"ff00",
		"ff00000",
		"gg0000",
		"ff0000:00ff00:0000ff",
		"ff0000:0",
		"",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, spec string) {
		mmss, hh, err := parseColorSpec(spec)
		if err != nil {
			return
		}
		// a single color must apply to both digit groups
		if strings.Count(spec, ":") == 0 {
			if mmss != hh {
				t.Fatalf("parseColorSpec(%q) single color gave mmss %+v != hh %+v", spec, mmss, hh)
			}
		}
	})
}

// FuzzDisplayModeString feeds arbitrary mode bytes to the display-mode
// decoder; every byte is valid input for the real clock, so the only
// invariant is that decoding never panics
func FuzzDisplayModeString(f *testing.F) {
	for _, seed := range []uint8{0x00, 0x01, 0x41, 0x02, 0x03, 0x04, 0x07, 0xC1, 0x61, 0xFF} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, mode uint8) {
		_ = displayModeString(mode)
	})
}

// FuzzStatusDecode feeds arbitrary byte packets to the API 1.x and 2.0
// decoders, exercising the paths the status subcommand takes when a
// clock (or something spoofing one) sends a malformed or hostile
// response: a 35-byte packet must decode as API 1.x, a 40-byte one as
// API 2.0, and neither may panic along the way
func FuzzStatusDecode(f *testing.F) {
	f.Add(r10Bytes())
	f.Add(make([]byte, 40))
	f.Add([]byte{})
	f.Add(make([]byte, 37))
	f.Fuzz(func(t *testing.T, packet []byte) {
		switch len(packet) {
		case api1PacketSize:
			var r10 Response10
			if err := binary.Read(bytes.NewReader(packet), binary.BigEndian, &r10); err != nil {
				t.Fatalf("decoding %d-byte packet as API 1.x: %v", len(packet), err)
			}
		case api2PacketSize:
			var r20 Response20
			if err := binary.Read(bytes.NewReader(packet), binary.BigEndian, &r20); err != nil {
				t.Fatalf("decoding %d-byte packet as API 2.0: %v", len(packet), err)
			}
		}
	})
}
