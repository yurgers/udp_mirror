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
		[]string{"pipeline_name", "sender", "lisneter_number"},
	)

	receivedBytesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "received_bytes_total",
			Help: "Total number of received bytes",
		},
		[]string{"pipeline_name", "sender", "lisneter_number"},
	)

	sentPacketsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sent_packets_total",
			Help: "Total number of sent packets",
		},
		[]string{"pipeline_name", "recipient"},
	)

	sendBytesCounter = prometheus.NewCounterVec(
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
	prometheus.MustRegister(receivedBytesCounter)

	prometheus.MustRegister(sentPacketsCounter)
	prometheus.MustRegister(sendBytesCounter)
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
func IncrementReceived(listName, plName, sender string, bytes int) {
	receivedPacketsCounter.WithLabelValues(plName, sender, listName).Inc()
	receivedBytesCounter.WithLabelValues(plName, sender, listName).Add(float64(bytes))
}

// IncrementSent увеличивает счетчик отправленных пакетов и байтов
func IncrementSent(plName, recipient string, bytes int) {
	sentPacketsCounter.WithLabelValues(plName, recipient).Inc()
	sendBytesCounter.WithLabelValues(plName, recipient).Add(float64(bytes))
}
