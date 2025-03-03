package sender_test

import (
	"bytes"
	"net"
	"testing"

	"golang.org/x/net/ipv4"
)

// func init() {
// 	src, _ := net.ResolveTCPAddr("upd", "1.1.1.1:12345")
// 	dst, _ := net.ResolveTCPAddr("upd", "1.1.1.1:12345")

// 	fmt.Print(src.IP, src.Port)
// 	fmt.Print(dst.IP, dst.Port)

// }

// var lenData int = 1500

// var udpHeader = []byte{
// 	byte(srcPort >> 8), byte(srcPort), // Src порт
// 	byte(dstPort >> 8), byte(dstPort), // Dst порт
// 	byte(lenData >> 8), byte(lenData), // Размер пакета (заголовок + данные)
// 	byte(0), byte(0), // Подставляем 0, для иннорирования контрольной сумму
// }

// var data100 = bytes.Repeat([]byte{0x01}, lenData)

// func copyMethod(data []byte) []byte {
// 	buffer := make([]byte, len(udpHeader)+len(data))
// 	copy(buffer, udpHeader)
// 	copy(buffer[len(udpHeader):], data)
// 	return buffer
// }

func BenchmarkRawConnSend(b *testing.B) {
	src := net.IPv4(1, 1, 1, 1)
	dst := net.IPv4(127, 0, 0, 1)

	// Создаем сырой сокет
	c, err := net.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		b.Fatalf("Failed to create packet conn: %v", err)
	}
	defer c.Close()

	// Создаем RawConn
	rc, err := ipv4.NewRawConn(c)
	if err != nil {
		b.Fatalf("Failed to create raw conn: %v", err)
	}
	defer rc.Close()

	// Подготовка тестовых данных
	// payload := []byte("benchmark-test-payload")
	payload := bytes.Repeat([]byte{0x01}, 2000)

	// Создаем IPv4 заголовок
	hdr := &ipv4.Header{
		Version:  4,
		Len:      ipv4.HeaderLen,
		TOS:      0x00,
		TotalLen: ipv4.HeaderLen + len(payload),
		TTL:      64,
		Protocol: 1, // ICMP
		Src:      src,
		Dst:      dst,
	}

	// Сбрасываем таймер перед основным циклом
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Обновляем идентификатор пакета для уникальности
		hdr.ID = i % 0xffff

		// Записываем пакет
		err := rc.WriteTo(hdr, payload, nil)
		if err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}
