package sender

import (
	"log"

	"golang.org/x/net/ipv4"

	"udp_mirror/config"
)

type UDPSender struct{}

func (s *UDPSender) SendPacket(rawConn *ipv4.RawConn, data []byte, src config.AddrConfig, dst config.AddrConfig) {
	header := ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TTL:      64,
		Protocol: 17, // UDP protocol
		Src:      src.Host,
		Dst:      dst.Host,
	}

	err := rawConn.WriteTo(&header, data, nil)
	if err != nil {
		log.Printf("Ошибка отправки пакета: %v", err)
	}
}
