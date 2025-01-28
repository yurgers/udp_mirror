// internal/manager/worker_manager.go
package manager

import (
	"udp_mirror/config"
	"udp_mirror/internal/sender"
	"udp_mirror/internal/worker"
)

type WorkerManager struct {
	Workers []*worker.Worker
	// wg      sync.WaitGroup
}

// NewWorkerManager создает и инициализирует WorkerManager
func NewWorkerManager(targets []config.TargetConfig, channels *[]chan worker.IRPData) *WorkerManager {
	manager := &WorkerManager{}

	for i, target := range targets {
		(*channels)[i] = make(chan worker.IRPData, 10)
		w := &worker.Worker{
			Target: target,
			Sender: &sender.UDPSender{},
		}
		manager.Workers = append(manager.Workers, w)

	}

	return manager
}

// startWorker запускает воркер
// func (m *WorkerManager) StartWorker(w *worker.Worker, c <-chan worker.IRPData) {
// 	defer m.wg.Done()

// 	log.Printf("Запуск воркера для %v", w)
// 	w.ProcessPackets(c)
// 	log.Printf("Воркер завершен для %v", w)
// }

// Shutdown завершает работу всех воркеров
// func (m *WorkerManager) Shutdown() {
// 	log.Println("Остановка всех воркеров")
// 	m.wg.Wait()
// }
