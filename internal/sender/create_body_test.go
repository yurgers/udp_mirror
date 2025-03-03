package sender_test

import (
	"bytes"
	"testing"
)

var srcPort uint16 = 12345
var dstPort uint16 = 12345
var lenData int = 1500

var udpHeader = []byte{
	byte(srcPort >> 8), byte(srcPort), // Src порт
	byte(dstPort >> 8), byte(dstPort), // Dst порт
	byte(lenData >> 8), byte(lenData), // Размер пакета (заголовок + данные)
	byte(0), byte(0), // Подставляем 0, для иннорирования контрольной сумму
}

var data100 = bytes.Repeat([]byte{0x01}, lenData)

// Метод 1: append с промежуточной копией
func appendMethod(data []byte) []byte {
	return append(append([]byte{}, udpHeader...), data...)
}

// Метод 2: make + copy
func copyMethod(data []byte) []byte {
	buffer := make([]byte, len(udpHeader)+len(data))
	copy(buffer, udpHeader)
	copy(buffer[len(udpHeader):], data)
	return buffer
}

// Бенчмарк для append
func BenchmarkAppendMethod(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = appendMethod(data100)
	}
}

// Бенчмарк для make + copy
func BenchmarkCopyMethod(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = copyMethod(data100)
	}
}
