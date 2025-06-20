package sender

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"

	"golang.org/x/net/ipv4"

	"udp_mirror/config"
	"udp_mirror/pkg/metrics"
)

var id uint16

type UDPSender struct {
	dst     config.AddrConfig
	mtu     int
	conn    net.PacketConn
	rawConn *ipv4.RawConn
	// mu      sync.Mutex
	plName string
}

func NewUDPSender(ctx context.Context, target config.TargetConfig) (PacketSender, error) {
	plName, _ := ctx.Value(config.PlNameKey).(string)

	listen := "127.0.0.1"
	if target.Host.String() != listen {
		listen = ""
	}

	conn, err := net.ListenPacket("ip4:udp", listen)
	if err != nil {
		return nil, err
	}

	rawConn, err := ipv4.NewRawConn(conn)
	if err != nil {
		return nil, err
	}

	dst := config.AddrConfig{
		Host: target.Host,
		Port: target.Port,
	}

	mtu := 1480 // 1500 - 20 (ip)  - 8 (udp)
	if dst.Host.String() == "127.0.0.1" {
		mtu = 65508
	}

	return &UDPSender{
		dst:     dst,
		mtu:     mtu,
		conn:    conn,
		rawConn: rawConn,
		plName:  plName,
	}, nil
}

func (s *UDPSender) SendPacket(data []byte, src config.AddrConfig) {
	id++

	// src.IP = net.IPv4(192, 168, 1, 78)

	udpHeader := []byte{
		byte(src.Port >> 8), byte(src.Port), // Src порт
		byte(s.dst.Port >> 8), byte(s.dst.Port), // Dst порт
		byte((len(data) + 8) >> 8), byte(len(data) + 8), // Размер пакета (заголовок + данные)
		byte(0), byte(0), // Подставляем 0, для иннорирования контрольной сумму
	}

	ipHeader := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TOS:      0x00,
		TotalLen: ipv4.HeaderLen + len(data) + 8,
		ID:       int(id),
		//Flags:    ipv4.DontFragment,
		FragOff:  0,
		TTL:      64,
		Protocol: 17,
		Src:      src.Host.To4(),
		Dst:      s.dst.Host.To4(),
	}

	buffer := make([]byte, len(udpHeader)+len(data))
	copy(buffer, udpHeader)
	copy(buffer[len(udpHeader):], data)
	// buffer := append(append([]byte{}, udpHeader...), data...)
	// log.Printf("Адрес buffer: %p\n", unsafe.Pointer(&buffer[0]))

	recipient := fmt.Sprintf("%s:%d", s.dst.Host.String(), s.dst.Port)
	metrics.IncrementSent(s.plName, recipient, len(data))

	if len(buffer) <= s.mtu {
		err := s.rawConn.WriteTo(ipHeader, buffer, nil)
		if err != nil {
			msg := fmt.Sprintf("[Pipeline %v] WriteTo %v: %v\n", s.plName, recipient, err)
			slog.Error(msg)
			return
		}

		// fmt.Println(iph)
		// fmt.Println("=====================")

	} else {
		// fmt.Println("MTU привешен", len(data), "len= ", len(buffer))
		fragOff := 0

		for {
			// fmt.Println("fragOff: ", fragOff)

			if len(buffer) > s.mtu {
				fragment := buffer[:s.mtu]
				// fmt.Println("len fragment ", len(fragment))
				buffer = buffer[s.mtu:]
				// fmt.Println("len new  buffer", len(buffer))
				ipHeader.Flags = ipv4.MoreFragments
				ipHeader.TotalLen = ipv4.HeaderLen + len(fragment)
				ipHeader.FragOff = fragOff
				fragOff += len(fragment) / 8

				// fmt.Printf("buffer: %v\n", buffer)

				err := s.rawConn.WriteTo(ipHeader, fragment, nil)
				if err != nil {
					msg := fmt.Sprintf("[Pipeline %v] WriteTo %v: %v\n", s.plName, recipient, err)
					slog.Error(msg)
					return
				}

			} else {
				ipHeader.Flags = 0
				ipHeader.FragOff = fragOff
				ipHeader.TotalLen = ipv4.HeaderLen + len(buffer)
				// fmt.Println("send finish buffer ", len(buffer))
				// fmt.Println("FragOff finish", ipHeader.FragOff)

				err := s.rawConn.WriteTo(ipHeader, buffer, nil)
				if err != nil {
					msg := fmt.Sprintf("[Pipeline %v] WriteTo %v: %v\n", s.plName, recipient, err)
					slog.Error(msg)
				}

				// fmt.Println(ipHeader)
				// fmt.Println("=====================")
				return
			}

			// fmt.Println(ipHeader)
			// fmt.Println("-------------------")

		}

	}
}

func (s *UDPSender) Close() {
	err := s.rawConn.Close()
	if err != nil {
		log.Fatalf("[Pipeline %v] err: %v\n", s.plName, err)
	}

	err = s.conn.Close()
	if err != nil {
		log.Fatalf("[Pipeline %v] err: %v\n", s.plName, err)
	}
}
