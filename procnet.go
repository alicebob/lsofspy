package procspy

import (
	"bytes"
	"net"
)

type ProcNet struct {
	b                       []byte
	c                       Connection
	wantedState             uint
	bytesLocal, bytesRemote [16]byte
}

func NewProcNet(b []byte, wantedState uint) *ProcNet {
	// Skip header
	b = nextLine(b)

	return &ProcNet{
		b:           b,
		c:           Connection{},
		wantedState: wantedState,
	}
}

func (p *ProcNet) Next() *Connection {
AGAIN:
	if len(p.b) == 0 {
		return nil
	}

	var (
		local, remote, state, inode []byte
	)
	_, p.b = nextField(p.b) // 'sl' column
	local, p.b = nextField(p.b)
	remote, p.b = nextField(p.b)
	state, p.b = nextField(p.b)
	if parseHex(state) != p.wantedState {
		p.b = nextLine(p.b)
		goto AGAIN
	}
	_, p.b = nextField(p.b) // 'tx_queue' column
	_, p.b = nextField(p.b) // 'rx_queue' column
	_, p.b = nextField(p.b) // 'tr' column
	_, p.b = nextField(p.b) // 'uid' column
	_, p.b = nextField(p.b) // 'timeout' column
	inode, p.b = nextField(p.b)

	p.c.LocalAddress, p.c.LocalPort = scanAddressNA(local, &p.bytesLocal)
	p.c.RemoteAddress, p.c.RemotePort = scanAddressNA(remote, &p.bytesRemote)
	p.c.inode = parseDec(inode)
	p.b = nextLine(p.b)
	return &p.c
}

// scanAddress parses 'A12CF62E:00AA' to the address/port. Handles IPv4 and
// IPv6 addresses.  The address is a big endian 32 bit ints, hex encoded. We
// just decode the hex and flip the bytes in every group of 4.
func scanAddressNA(in []byte, buf *[16]byte) (net.IP, uint16) {
	col := bytes.IndexByte(in, ':')
	if col == -1 {
		return nil, 0
	}

	// Network address is big endian. Can be either ipv4 or ipv6.
	address := hexDecode32bigNA(in[:col], buf)
	return net.IP(address), uint16(parseHex(in[col+1:]))
}

// hexDecode32big decodes sequences of 32bit big endian bytes.
func hexDecode32bigNA(src []byte, buf *[16]byte) []byte {
	blocks := len(src) / 8
	for block := 0; block < blocks; block++ {
		for i := 0; i < 4; i++ {
			a := fromHexChar(src[block*8+i*2])
			b := fromHexChar(src[block*8+i*2+1])
			buf[block*4+3-i] = (a << 4) | b
		}
	}
	return buf[:blocks*4]
}
