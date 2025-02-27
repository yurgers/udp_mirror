package manager

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"udp_mirror/config"
	"udp_mirror/internal/sender"
	"udp_mirror/internal/worker"
)

type WorkerManager struct {
	Workers []*worker.Worker
	count   int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

type SenderFactoryFunc func(context.Context, config.TargetConfig) (sender.PacketSender, error)

// NewWorkerManager создает и инициализирует WorkerManager
func NewWorkerManager(ctx context.Context, targets []config.TargetConfig, senderFactory SenderFactoryFunc) (*WorkerManager, error) {
	ctx, cancel := context.WithCancel(ctx)
	count := runtime.NumCPU() / 2
	// count := 8

	manager := &WorkerManager{
		count:  count,
		ctx:    ctx,
		cancel: cancel,
	}

	for _, target := range targets {
		for range count {
			// Создаем менеджер воркеров
			sender, err := senderFactory(ctx, target)
			if err != nil {
				return nil, fmt.Errorf("ошибка при создании PacketSender: %w", err)
			}

			w := &worker.Worker{
				Target: target,
				Sender: sender,
			}
			manager.Workers = append(manager.Workers, w)
		}
	}

	return manager, nil
}

func (wm *WorkerManager) Start(chs []chan worker.IRPData) {
	for i, wk := range wm.Workers {
		wm.wg.Add(1)
		go func(w *worker.Worker, ch <-chan worker.IRPData) {
			defer wm.wg.Done()
			w.StartProcessPackets(wm.ctx, ch)
		}(wk, chs[i/wm.count])
	}
}

func (wm *WorkerManager) Shutdown() {
	wm.cancel()
	wm.wg.Wait()
}
