// Go implementation of port knocking.
package knock

import (
	"bytes"
	"io"
	"net"
)

// A single knock to either listen for or execute.
type Knock struct {
	// port to knock/listen on
	Port uint16
	// bytes to send
	Payload []byte
}

func Port(number uint16) Knock {
	return Knock{
		Port:    number,
		Payload: []byte{},
	}
}

func PayLoad(port uint16, payload []byte) Knock {
	return Knock{
		Port:    port,
		Payload: payload,
	}
}

// Sends knocks to host
func Send(host string, knocks ...Knock) error {

	conn, err := net.Dial("ip4:tcp", host)

	if err != nil {
		return err
	}
	defer conn.Close()

	for _, knock := range knocks {
		packet := new(KnockPacket)

		packet.SourcePort = 0x1337
		packet.DestinationPort = knock.Port

		packet.Data = knock.Payload

		buffers := makeChecksumedBuffers(conn, packet)

		var pkBuffer *bytes.Buffer = new(bytes.Buffer)

		buffers.WriteTo(pkBuffer)
		pkBuffer.Write([]byte{0x00, 0x00})
		pkBuffer.Write(knock.Payload)

		pkBuffer.WriteTo(conn)

	}
	return nil
}

type KnockPacket struct {
	SourcePort      uint16
	DestinationPort uint16
	SeqNumber       uint32
	AckNumber       uint32
	Window          uint16
	Data            []byte
}

func u16(n uint16) []byte {
	return []byte{byte(n >> 8), byte(n & 0xff)}
}

func u32(n uint32) []byte {
	return []byte{
		byte(n >> 24), byte((n & 0x00ff0000) >> 16),
		byte(n >> 8), byte((n & 0xff)),
	}
}

func (packet *KnockPacket) Bytes() []byte {
	var buffer *bytes.Buffer = new(bytes.Buffer)

	buffer.Write(u16(packet.SourcePort))
	buffer.Write(u16(packet.DestinationPort))

	buffer.Write(u32(packet.SeqNumber))
	buffer.Write(u32(packet.AckNumber))

	// Data Offset + Reserved + Control bits
	// Here we are setting the SYN bit only
	buffer.Write([]byte{(20 << 2), 0x02})

	buffer.Write(u16(packet.Window))

	buffer.Write(packet.Data)

	return buffer.Bytes()
}

func (packet *KnockPacket) Buffers() net.Buffers {
	// From RFC 793
	//
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |          Source Port          |       Destination Port        |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                        Sequence Number                        |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                    Acknowledgment Number                      |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |  Data |           |U|A|P|R|S|F|                               |
	// | Offset| Reserved  |R|C|S|S|Y|I|            Window             |
	// |       |           |G|K|H|T|N|N|                               |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |           Checksum            |         Urgent Pointer        |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                    Options                    |    Padding    |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |                             data                              |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	var buffers net.Buffers = [][]byte{
		u16(packet.SourcePort), u16(packet.DestinationPort),

		u32(packet.SeqNumber),
		u32(packet.AckNumber),

		[]byte{(20 << 2), 0x02}, u16(packet.Window),
		[]byte{0x00, 0x00},
	}
	return buffers
}

func makePsuedoHeader(conn net.Conn, tcplen uint16) net.Buffers {
	// +--------+--------+--------+--------+
	// |           Source Address          |
	// +--------+--------+--------+--------+
	// |         Destination Address       |
	// +--------+--------+--------+--------+
	// |  zero  |  PTCL  |    TCP Length   |
	// +--------+--------+--------+--------+
	var buffers net.Buffers = [][]byte{
		net.ParseIP(conn.LocalAddr().String()),
		net.ParseIP(conn.RemoteAddr().String()),
		// zero + PTCL (6 is the tcp protocal number)
		[]byte{0x00, 0x06}, u16(tcplen),
	}
	return buffers
}

func calculateChecksum(buffers io.Reader) uint16 {
	var buffer []byte = make([]byte, 2)
	var checksum uint64
	for {
		_, err := buffers.Read(buffer)
		checksum += ^(uint64((buffer[0] << 8)) + uint64(buffer[1]))
		if err == io.EOF {
			break
		}
	}
	for (checksum >> 16) == 0 {
		checksum = (checksum & 0xffff) + (checksum >> 16)
	}
	return uint16(^checksum)
}

func makeChecksumedBuffers(conn net.Conn, packet *KnockPacket) net.Buffers {
	buffers := packet.Buffers()
	var tcplen uint16
	for _, buf := range buffers {
		tcplen += uint16(len(buf))
	}
	pseudoHeader := makePsuedoHeader(conn, tcplen)
	checksumBuffers := append(pseudoHeader, buffers...)

	checksum := calculateChecksum(&checksumBuffers)
	// I want to find a better way of setting the checksum here
	// The magic indexing doesn't seem like the best
	buffers[6] = u16(checksum)
	return buffers
}
