package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"udp_mirror/config"
	"udp_mirror/internal/listener"
	"udp_mirror/internal/manager"
	"udp_mirror/internal/metrics"
	"udp_mirror/internal/pprof"
	"udp_mirror/internal/worker"
)

func main() {
	// Парсим флаги
	configFilePtr := flag.String("f", "config.yml", "Path to the config file")
	backgroundFlag := flag.Bool("d", false, "Run in background mode")
	flag.Parse()

	// Загружаем конфигурацию
	cfg := config.GetConfig(*configFilePtr)
	fmt.Println(cfg)

	if *backgroundFlag {
		runInBackground(*configFilePtr, flag.Args())
	}

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

	// Создаем слушателя
	channels := make([]chan worker.IRPData, len(cfg.Targets))
	listener, err := listener.NewUDPListener(cfg.Server, channels)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// Создаем менеджер воркеров
	workerManager := manager.NewWorkerManager(cfg.Targets, &channels)

	var wg sync.WaitGroup
	for i, w := range workerManager.Workers {
		wg.Add(1)

		go func(worker *worker.Worker, ch <-chan worker.IRPData) {
			defer wg.Done()
			worker.ProcessPackets(ch)
			log.Printf("Воркер завершен для %v\n", w)
		}(w, channels[i])

	}

	fmt.Printf("Сервер запущен и слушает на %s\n", listener.LocalAddr())

	go func() {
		listener.Listen()
		defer listener.Shutdown()
	}()

	wg.Wait()
}

func runInBackground(configFile string, args []string) {
	cmdArgs := append([]string{"-f", configFile}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Start()
	os.Exit(0)
}
