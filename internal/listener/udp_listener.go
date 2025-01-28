package listener

import (
	"fmt"
	"log"
	"net"

	"udp_mirror/config"
	"udp_mirror/internal/worker"
)

type UDPListener struct {
	conn     *net.UDPConn
	channels []chan worker.IRPData
}

// NewUDPListener создает новый экземпляр UDPListener
func NewUDPListener(serverAddr config.AddrConfig, channels []chan worker.IRPData) (*UDPListener, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", serverAddr.Host.String(), serverAddr.Port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	return &UDPListener{
		conn:     conn,
		channels: channels,
	}, nil
}

// Listen начинает прием данных с UDP-соединения
func (l *UDPListener) Listen() {
	buffer := make([]byte, 65536)

	for {
		n, src, err := l.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Ошибка чтения из UDP: %v", err)
			continue
		}

		l.processData(buffer[:n], src)
	}
}

// processData обрабатывает полученные данные и отправляет их в каналы
func (l *UDPListener) processData(data []byte, src *net.UDPAddr) {
	for _, channel := range l.channels {
		channel <- worker.IRPData{
			Data: data,
			Src: config.AddrConfig{
				Host: src.IP,
				Port: uint16(src.Port),
			},
		}
	}
}

func (l *UDPListener) Shutdown() {
	for _, channel := range l.channels {
		close(channel)
	}
}

// Close закрывает соединение UDP
func (l *UDPListener) Close() error {
	return l.conn.Close()
}

// LocalAddr возвращает локальный адрес соединения
func (l *UDPListener) LocalAddr() net.Addr {
	return l.conn.LocalAddr()
}
