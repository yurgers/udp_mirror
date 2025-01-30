package manager

import (
	"context"
	"errors"
	"sync"

	"udp_mirror/config"
	"udp_mirror/internal/sender"
	"udp_mirror/internal/worker"
)

type WorkerManager struct {
	Workers []*worker.Worker

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

type SenderFactoryFunc func(config.TargetConfig) (sender.PacketSender, error)

// NewWorkerManager создает и инициализирует WorkerManager
func NewWorkerManager(ctx context.Context, targets []config.TargetConfig, senderFactory SenderFactoryFunc) (*WorkerManager, error) {
	ctx, cancel := context.WithCancel(ctx)

	manager := &WorkerManager{
		ctx:    ctx,
		cancel: cancel,
	}

	for _, target := range targets {
		// Создаем менеджер воркеров
		sender, err := senderFactory(target)
		if err != nil || sender == nil {
			return nil, errors.New("ошибка при создании PacketSender")
		}

		w := &worker.Worker{
			Target: target,
			Sender: sender,
		}
		manager.Workers = append(manager.Workers, w)

	}

	return manager, nil
}

func (wm *WorkerManager) Start(chs []chan worker.IRPData) {
	for i, wk := range wm.Workers {
		wm.wg.Add(1)
		go func(w *worker.Worker, ch <-chan worker.IRPData) {
			defer wm.wg.Done()
			w.ProcessPackets(ch)
		}(wk, chs[i])
	}
}

func (wm *WorkerManager) Shutdown() {
	wm.wg.Wait()
	wm.cancel()
}
