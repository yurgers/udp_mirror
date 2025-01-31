package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"udp_mirror/config"
	"udp_mirror/internal/pipeline"

	"udp_mirror/pkg/metrics"
	"udp_mirror/pkg/pprof"
)

func main() {
	// Парсим флаги
	configFilePtr := flag.String("f", "config.yml", "Path to the config file")
	backgroundFlag := flag.Bool("d", false, "Run in background mode")
	signalFlag := flag.String("s", "", "Send signal to running process (reload, stop, quit)")
	flag.Parse()

	if *signalFlag == "reload" {
		sendReloadSignal()
		return
	} else if *signalFlag == "quit" {
		sendQuitSignal()
		return
	} else if *signalFlag != "" {
		flag.Usage()
		os.Exit(1)
	}

	if *backgroundFlag {
		runInBackground(*configFilePtr, flag.Args())
	}

	writePIDFile()
	defer deletePIDFile()

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

func writePIDFile() {
	myFilePid := "udp_mirror.pid"
	pid := os.Getpid()

	_, err := os.Stat(myFilePid)
	if err == nil {
		log.Fatalln("Просесс udp_mirror уже запущен")
	} else {
		if !os.IsNotExist(err) {
			log.Fatalf("Ошибка c pid файлом, %s\n", err)
		}

	}

	err = os.WriteFile(myFilePid, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		log.Fatalf("Ошибка записи PID файла: %v", err)
	}
}

func deletePIDFile() {
	err := os.Remove("udp_mirror.pid")
	if err != nil {
		log.Fatalln(err)
	}
}

func sendQuitSignal() {
	pid, err := os.ReadFile("udp_mirror.pid") // Читаем PID из файла
	if err != nil {
		log.Fatalf("Ошибка чтения PID файла: %v", err)
	}

	pidInt, err := strconv.Atoi(string(pid))
	if err != nil {
		log.Fatalf("Ошибка конвертации PID: %v", err)
	}

	err = syscall.Kill(pidInt, syscall.SIGTERM)
	if err != nil {
		log.Fatalf("Ошибка отправки сигнала: %v", err)
	}

	log.Println("Приложение завершено!")
}

func sendStopSignal() {
	pid, err := os.ReadFile("udp_mirror.pid") // Читаем PID из файла
	if err != nil {
		log.Fatalf("Приложение не запущено")
	}

	pidInt, err := strconv.Atoi(string(pid))
	if err != nil {
		log.Fatalf("Ошибка конвертации PID: %v", err)
	}

	err = syscall.Kill(pidInt, syscall.SIGKILL)
	if err != nil {
		log.Fatalf("Ошибка отправки сигнала: %v", err)
	}

	log.Println("Приложение принудительно завершено!")
}

func sendReloadSignal() {
	log.Println("Получен сигнал на перезапуск...")
	pid, err := os.ReadFile("udp_mirror.pid") // Читаем PID из файла
	if err != nil {
		log.Fatalf("Приложение не запущено")
	}

	pidInt, err := strconv.Atoi(string(pid))
	if err != nil {
		log.Fatalf("Ошибка конвертации PID: %v", err)
	}

	err = syscall.Kill(pidInt, syscall.SIGHUP) // Отправляем `SIGHUP`
	if err != nil {
		log.Fatalf("Ошибка отправки сигнала: %v", err)
	}

	log.Println("Конфиг успешно перезагружается!")
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
