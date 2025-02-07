package pipeline

import (
	"context"
	"log"
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

// NewWorkerManager создает и инициализирует WorkerManager
func NewPipeline(p_cfg config.Pipeline) *Pipeline {

	chs := make([]chan worker.IRPData, len(p_cfg.Targets))
	//инциализация chs
	for i := range p_cfg.Targets {
		chs[i] = make(chan worker.IRPData, 20)
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
		log.Fatalf("[Pipeline %s] Ошибка запуска UDP слушателя: %v\n", pl.Name, err)
	}
	defer listener.Close()

	go func() {
		listener.Start()
		defer listener.Shutdown()
	}()

	workerManager, err := manager.NewWorkerManager(ctx, pl.Targets, sender.NewUDPSender)
	if err != nil {
		log.Panicf("[Pipeline %s] %v", pl.Name, err)
	}

	workerManager.Start(pl.Channels)

	<-ctx.Done()
	log.Printf("[Pipeline %s] Остановка...\n", pl.Name)
	workerManager.Shutdown()

	log.Printf("[Pipeline %s] Завершен\n", pl.Name)
}
