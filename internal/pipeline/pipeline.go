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

	ctx    context.Context
	cancel context.CancelFunc
}

// NewWorkerManager создает и инициализирует WorkerManager
func NewPipeline(ctx context.Context, p_cfg config.Pipeline) *Pipeline {
	ctx, cancel := context.WithCancel(ctx)

	chs := make([]chan worker.IRPData, len(p_cfg.Targets))
	//инциализация chs
	for i := range p_cfg.Targets {
		chs[i] = make(chan worker.IRPData, 10)
	}

	pipeline := &Pipeline{
		Channels: chs,

		Name:    p_cfg.Name,
		Input:   p_cfg.Input,
		Targets: p_cfg.Targets,

		ctx:    ctx,
		cancel: cancel,
	}

	return pipeline
}

func (pl *Pipeline) Start(ctx context.Context) {
	log.Printf("Запуск pipeline %s...\n", pl.Name)

	// Создаем слушателя
	listener, err := listener.NewUDPListener(ctx, pl.Input, pl.Channels)
	if err != nil {
		log.Fatalf("Pipeline %s. Ошибка запуска UDP слушателя: %v\n", pl.Name, err)
	}
	defer listener.Close()

	go func() {
		listener.Start()
		defer listener.Shutdown()
	}()

	workerManager, err := manager.NewWorkerManager(pl.ctx, pl.Targets, sender.NewUDPSender)
	if err != nil {
		log.Panicln(err)
	}

	workerManager.Start(pl.Channels)

	log.Printf("Сервер %s запущен и слушает на %s\n", pl.Name, listener.LocalAddr())

	<-pl.ctx.Done()

	log.Printf("Остановка pipeline %s...\n", pl.Name)
	workerManager.Shutdown()

	pl.cancel()

}
