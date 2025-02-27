package pipeline

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"runtime"
	"sync"
	"udp_mirror/config"
	"udp_mirror/internal/listener"
	"udp_mirror/internal/manager"
	"udp_mirror/internal/sender"
	"udp_mirror/internal/worker"
)

type Pipeline struct {
	Channels []chan worker.IRPData

	Name    string
	Input   config.AddrConfig
	Targets []config.TargetConfig
}

// NewPipeline создает и инициализирует Pipeline
func NewPipeline(p_cfg config.Pipeline) *Pipeline {

	chs := make([]chan worker.IRPData, len(p_cfg.Targets))

	// инциализация chs
	for i := range p_cfg.Targets {
		chs[i] = make(chan worker.IRPData, 1500)
	}

	pipeline := &Pipeline{
		Channels: chs,

		Name:    p_cfg.Name,
		Input:   p_cfg.Input,
		Targets: p_cfg.Targets,
	}

	return pipeline
}

func (pl *Pipeline) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx = context.WithValue(ctx, config.PlNameKey, pl.Name)

	log.Printf("[Pipeline %s] Запуск...\n", pl.Name)

	// Создаем слушателя
	listener, err := listener.NewUDPListener(ctx, pl.Input, pl.Channels)
	if err != nil {
		msg := fmt.Sprintf("[Pipeline %s] Ошибка запуска UDP слушателя: %v\n", pl.Name, err)
		slog.Error(msg)
		return
	}
	// defer listener.Close()
	listenerWorker := runtime.NumCPU() / 2
	// listenerWorker := 4
	var wg sync.WaitGroup

	for i := 0; i < listenerWorker; i++ {
		lName := fmt.Sprintf("%d", i)
		wg.Add(1)
		go func(lName string) {
			listener.Start(lName)
			defer wg.Done()
		}(lName)
	}
	workerManager, err := manager.NewWorkerManager(ctx, pl.Targets, sender.NewUDPSender)
	if err != nil {
		log.Panicf("[Pipeline %s] %v", pl.Name, err)
	}

	workerManager.Start(pl.Channels)

	<-ctx.Done()
	log.Printf("[Pipeline %s] Остановка...\n", pl.Name)

	wg.Wait()
	listener.Shutdown()

	workerManager.Shutdown()

	log.Printf("[Pipeline %s] Завершен\n", pl.Name)
}
