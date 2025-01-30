package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"udp_mirror/config"
	"udp_mirror/internal/metrics"
	"udp_mirror/internal/pipeline"
	"udp_mirror/internal/pprof"
)

func main() {
	// Парсим флаги
	configFilePtr := flag.String("f", "config.yml", "Path to the config file")
	backgroundFlag := flag.Bool("d", false, "Run in background mode")
	flag.Parse()

	if *backgroundFlag {
		runInBackground(*configFilePtr, flag.Args())
	}

	// Загружаем конфигурацию
	cfg := config.GetConfig(*configFilePtr)
	log.Println("Config:", cfg)

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Гарантируем отмену контекста при выходе

	go handleShutdown(cancel)

	// Регистрируем метрики
	metrics.Register()

	// Запускаем pprof, если включено
	if cfg.Pprof != nil && cfg.Pprof.Enabled {
		go pprof.Start(cfg.Pprof.Listen)
	}

	// Запускаем Prometheus, если включено
	if cfg.Prom != nil && cfg.Prom.Enabled {
		go metrics.StartPrometheus(cfg.Prom.Listen)
	}

	// p_cfg := cfg.Pipeline[0]
	// pl := pipeline.NewPipeline(ctx, p_cfg)
	// pl.Start(ctx)

	var wg_pl sync.WaitGroup

	for _, pl_cfg := range cfg.Pipeline {
		wg_pl.Add(1)
		go func(p_cfg config.Pipeline) {
			defer wg_pl.Done()
			pl := pipeline.NewPipeline(p_cfg)
			pl.Start(ctx)
		}(pl_cfg)
	}

	// Ожидаем завершения контекста (когда вызовем cancel)
	<-ctx.Done()
	log.Println("[Main] Остановка сервера...")

	wg_pl.Wait()
	log.Println("[Main] Все Pipeline завершены...")

	// workerManager.Shutdown()
	log.Println("[Main] Сервер завершил работу")
}

func runInBackground(configFile string, args []string) {
	cmdArgs := append([]string{"-f", configFile}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Start()
	os.Exit(0)
}

// handleShutdown ловит SIGINT/SIGTERM и вызывает cancel()
func handleShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan // Ждём сигнала
	log.Println("[Main] Получен сигнал завершения, останавливаем сервер...")
	cancel() // Отправляем сигнал остановки всем горутинам
}
