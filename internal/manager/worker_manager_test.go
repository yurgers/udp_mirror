// internal/manager/worker_manager_test.go
package manager

import (
	"testing"
	"time"

	"udp_mirror/config"
	"udp_mirror/internal/worker"
)

type MockSender struct {
	calls int
}

func (m *MockSender) SendPacket(_, _ interface{}, _, _ config.AddrConfig) {
	m.calls++
}

func TestWorkerManager(t *testing.T) {
	channels := []chan worker.IRPData{make(chan worker.IRPData, 1)}
	target := config.AddrConfig{Host: []byte{127, 0, 0, 1}, Port: 1234}

	manager := NewWorkerManager([]config.AddrConfig{target}, channels)
	defer manager.Shutdown()

	mockSender := &MockSender{}
	manager.workers[0].Sender = mockSender

	// Отправляем тестовые данные
	data := worker.IRPData{
		Data: []byte("test"),
		Src:  worker.AddrConfig{Host: []byte{127, 0, 0, 1}, Port: 5678},
	}
	channels[0] <- data
	close(channels[0])

	time.Sleep(100 * time.Millisecond)

	if mockSender.calls != 1 {
		t.Errorf("Ожидалось 1 вызов SendPacket, получено %d", mockSender.calls)
	}
}
