package sender

import (
	"udp_mirror/config"
)

// PacketSender определяет интерфейс для отправки UDP-пакетов
type PacketSender interface {
	SendPacket(data []byte, src config.AddrConfig)
	Close()
}
