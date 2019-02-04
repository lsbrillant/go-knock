// Go implementation of port knocking.
package knock

import (
	"bytes"
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

	for _, knock := range knocks {
		conn, err := net.Dial("ip4:tcp", host)
		if err != nil {
			return err
		}
		defer conn.Close()
		// Ok so now we want to make a TCP SYN packet
		//
		// From RFC 793
		//
		// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
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
			// Local addr shouldn't matter because we are closing
			// the connection affter sending a SYN
			[]byte{0x13, 0x37},
			// This is the dest port
			[]byte{byte(knock.Port >> 8), byte(knock.Port & 0x00ff)},
			// Sequence Number
			[]byte{0x00, 0x00}, []byte{0x00, 0x00},
			// Acknowledgment Number
			[]byte{0x00, 0x00}, []byte{0x00, 0x00},
			// Data Offset? + Reserved + Control bits
			// Here we are setting the SYN bit only
			[]byte{(20 << 2), 0x02},
			// Window
			[]byte{0x00, 0x00},
			// Checksum placeholder
			[]byte{0x00, 0x00},
		}
		// TCP pseudo Header for calculatingthe checksum
		var pseudoHeader net.Buffers = [][]byte{
			net.ParseIP(conn.LocalAddr().String()),
			net.ParseIP(conn.RemoteAddr().String()),
		}

		var i uint16
		// Calculate the Checksum
		var checksum uint16
		// local addr
		i = (uint16(pseudoHeader[0][0]) << 8) + uint16(pseudoHeader[0][1])
		checksum = checksum + (i ^ 0xffff)
		i = (uint16(pseudoHeader[0][2]) << 8) + uint16(pseudoHeader[0][3])
		checksum = checksum + (i ^ 0xffff)
		// remote addr
		i = (uint16(pseudoHeader[1][0]) << 8) + uint16(pseudoHeader[1][1])
		checksum = checksum + (i ^ 0xffff)
		i = (uint16(pseudoHeader[1][2]) << 8) + uint16(pseudoHeader[1][3])
		checksum = checksum + (i ^ 0xffff)
		// zero + PTCL
		checksum = checksum + (0x0006 ^ 0xffff)

		checksum = checksum + ((0x0012 + uint16(len(knock.Payload))) ^ 0xffff)

		for _, buff := range buffers {
			i = (uint16(buff[0]) << 8) + uint16(buff[1])
			checksum = checksum + (i ^ 0xffff)
		}
		checksum = checksum ^ 0xffff

		// Set the checksum
		buffers[len(buffers)-1][0] = byte(checksum >> 8)
		buffers[len(buffers)-1][1] = byte(checksum & 0x00ff)

		var pkBuffer *bytes.Buffer = new(bytes.Buffer)

		buffers.WriteTo(pkBuffer)
		pkBuffer.Write([]byte{0x00, 0x00})
		pkBuffer.Write(knock.Payload)

		pkBuffer.WriteTo(conn)

	}
	return nil
}
