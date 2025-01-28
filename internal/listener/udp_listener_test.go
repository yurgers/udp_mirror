// internal/listener/udp_listener_test.go
package listener

import (
	"net"
	"testing"
	"time"

	"udp_mirror/internal/worker"
)

func TestUDPListener(t *testing.T) {
	serverAddr := "127.0.0.1:9876"
	channels := []chan worker.IRPData{make(chan worker.IRPData, 1)}

	listener, err := NewUDPListener(serverAddr, channels)
	if err != nil {
		t.Fatalf("Ошибка создания UDPListener: %v", err)
	}
	defer listener.Close()

	go listener.Listen()

	// Отправляем тестовый пакет
	clientAddr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, err := net.DialUDP("udp", clientAddr, listener.conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatalf("Ошибка подключения клиента: %v", err)
	}
	defer conn.Close()

	message := []byte("test")
	_, err = conn.Write(message)
	if err != nil {
		t.Fatalf("Ошибка отправки данных: %v", err)
	}

	select {
	case data := <-channels[0]:
		if string(data.Data) != "test" {
			t.Errorf("Ожидалось 'test', получено '%s'", string(data.Data))
		}
	case <-time.After(time.Second):
		t.Errorf("Таймаут ожидания данных")
	}
}
