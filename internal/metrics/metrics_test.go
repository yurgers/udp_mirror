// internal/metrics/metrics_test.go
package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestMetrics(t *testing.T) {
	Register()

	IncrementReceived("sender1", 1024)
	IncrementSent("recipient1", 512)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "received_packets_total{sender=\"sender1\"} 1") {
		t.Errorf("Metrics not incremented for received packets")
	}

	if !strings.Contains(body, "sent_bytes_total{recipient=\"recipient1\"} 512") {
		t.Errorf("Metrics not incremented for sent bytes")
	}
}
