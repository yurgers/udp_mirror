package metrics

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	receivedPacketsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "received_packets_total",
			Help: "Total number of received packets",
		},
		[]string{"pipeline_name", "sender"},
	)

	bytesReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "received_bytes_total",
			Help: "Total number of received bytes",
		},
		[]string{"pipeline_name", "sender"},
	)

	sentPacketsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sent_packets_total",
			Help: "Total number of sent packets",
		},
		[]string{"pipeline_name", "recipient"},
	)

	bytesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sent_bytes_total",
			Help: "Total number of sent bytes",
		},
		[]string{"pipeline_name", "recipient"},
	)
)

// Register регистрирует метрики в Prometheus
func Register() {
	prometheus.MustRegister(receivedPacketsCounter)
	prometheus.MustRegister(bytesReceived)
	prometheus.MustRegister(sentPacketsCounter)
	prometheus.MustRegister(bytesSent)
}

// StartPrometheus запускает сервер для экспорта метрик
func StartPrometheus(addr string) {
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Prometheus метрики доступны на %s/metrics", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Ошибка запуска Prometheus: %v", err)
	}
}

// IncrementReceived увеличивает счетчик полученных пакетов и байтов
func IncrementReceived(plName string, sender string, bytes int) {
	receivedPacketsCounter.WithLabelValues(plName, sender).Inc()
	bytesReceived.WithLabelValues(plName, sender).Add(float64(bytes))
}

// IncrementSent увеличивает счетчик отправленных пакетов и байтов
func IncrementSent(plName string, recipient string, bytes int) {
	sentPacketsCounter.WithLabelValues(plName, recipient).Inc()
	bytesSent.WithLabelValues(plName, recipient).Add(float64(bytes))
}
