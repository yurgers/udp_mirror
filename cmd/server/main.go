package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"udp_mirror/config"
	"udp_mirror/internal/pipeline"

	"udp_mirror/pkg/metrics"
	"udp_mirror/pkg/pprof"
)

func initLog() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	// Создаем логгер с обычным текстовым выводом
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	// Делаем этот логгер дефолтным
	slog.SetDefault(logger)
}

func main() {
	initLog()

	// Парсим флаги
	configFilePtr := flag.String("f", "config.yml", "Path to the config file")
	backgroundFlag := flag.Bool("d", false, "Run in background mode")
	signalFlag := flag.String("s", "", "Send signal to running process (reload, stop, quit)")
	flag.Parse()

	slog.Debug(fmt.Sprintf("Используемый config файл: %s", *configFilePtr))
	pidFile := generatePIDFileName(*configFilePtr)

	if *signalFlag == "reload" {
		sendReloadSignal(pidFile)
		return
	} else if *signalFlag == "quit" {
		sendQuitSignal(pidFile)
		return
	} else if *signalFlag == "stop" {
		sendStopSignal(pidFile)
		return
	} else if *signalFlag != "" {
		flag.Usage()
		os.Exit(1)
	}

	if *backgroundFlag {
		runInBackground(*configFilePtr, flag.Args())
	}

	writePIDFile(pidFile)
	defer deletePIDFile(pidFile)

	// Загружаем конфигурацию
	// cfg := config.GetConfig(*configFilePtr)

	reloader := &config.ConfigReloader{}
	err := reloader.LoadConfig(*configFilePtr)
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
	}

	cfg := reloader.GetConfigCopy()
	log.Println("Config:", cfg)

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Гарантируем отмену контекста при выходе

	go handleShutdown(cancel)
	go handleReload(reloader, *configFilePtr)

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

func generatePIDFileName(configPath string) string {
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		log.Fatalf("Ошибка получения абсолютного пути: %v", err)
	}

	// Вариант 2: MD5-хэш пути (если путь слишком длинный)
	hash := md5.Sum([]byte(absPath))
	return "udp_mirror_" + hex.EncodeToString(hash[:8]) + ".pid"
}

func processExists(pidFile string) error {

	_, err := os.Stat(pidFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("Ошибка c pid файлом, %s\n", err)
		}
		return nil
	}

	pid := readPidFile(pidFile)

	process, _ := os.FindProcess(pid)
	if process == nil {
		deletePIDFile(pidFile)
	}

	return nil

}

func writePIDFile(pidFile string) {
	pid := os.Getpid()

	err := processExists(pidFile)
	if err != nil {
		log.Fatalln("Просесс udp_mirror уже запущен")
	}

	err = os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		log.Fatalf("Ошибка записи PID файла: %v", err)
	}
}

func deletePIDFile(pidFile string) {
	err := os.Remove(pidFile)
	if err != nil {
		log.Fatalln(err)
	}
}

func readPidFile(pidFile string) int {
	pid, err := os.ReadFile(pidFile) // Читаем PID из файла
	if err != nil {
		log.Fatalf("Ошибка чтения PID файла: %v", err)
	}

	cleaned := strings.TrimSpace(string(pid))
	pidInt, err := strconv.Atoi(string(cleaned))
	if err != nil {
		log.Fatalf("Ошибка конвертации PID: %v", err)
	}

	return pidInt
}

func sendQuitSignal(pidFile string) {
	pidInt := readPidFile(pidFile)

	err := syscall.Kill(pidInt, syscall.SIGTERM)
	if err != nil {
		log.Fatalf("Ошибка отправки сигнала: %v", err)
	}

	log.Println("Приложение завершено!")
}

func sendStopSignal(pidFile string) {
	defer deletePIDFile(pidFile)
	pid, err := os.ReadFile(pidFile) // Читаем PID из файла
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

func sendReloadSignal(pidFile string) {
	log.Println("Получен сигнал на перезапуск...")
	pid, err := os.ReadFile(pidFile) // Читаем PID из файла
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
	err := cmd.Start()
	if err != nil {
		log.Fatalln(err)
	}
	os.Exit(0)
}

// handleShutdown ловит SIGINT/SIGTERM и вызывает cancel()
func handleShutdown(cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan // Ждём SIGINT (Ctrl+C) или SIGTERM (kill <PID>)
	log.Println("[Main] Получен сигнал завершения, останавливаем сервер...")
	cancel() // Отправляем сигнал остановки всем горутинам
}

func handleReload(reloader *config.ConfigReloader, configFile string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	for sig := range sigChan {
		if sig == syscall.SIGHUP {
			log.Println("[Config] Получен SIGHUP, обновляем конфиг...")
			err := reloader.LoadConfig(configFile)
			if err != nil {
				log.Printf("Ошибка обновления конфига: %v", err)
			}

			cfg := reloader.GetConfigCopy()
			log.Println("Config:", cfg)

		}
	}
}
