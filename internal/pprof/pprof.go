// internal/pprof/pprof.go
package pprof

import (
	"log"
	"net/http"
	"net/http/pprof"
)

// Start запускает pprof-сервер на указанном адресе
func Start(addr string) {
	mux := http.NewServeMux()

	// Регистрация pprof-обработчиков
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	log.Printf("pprof запущен на %s/debug/pprof/", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Ошибка запуска pprof: %v", err)
	}
}
