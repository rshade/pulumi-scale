package routers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rshade/pulumi-scale/internal/webhooks"
)

func TestPrometheusHandler(t *testing.T) {
	intentChan := make(chan webhooks.ScalingIntent, 10) // Buffer for multiple alerts
	handler := PrometheusHandler(intentChan)

	r := chi.NewRouter()
	r.Post("/webhook/{pool}/prometheus", handler)

	// Test case: Alert with matching pool label
	alertPayload := struct {
		Alerts []struct {
			Labels map[string]string `json:"labels"`
			Status string            `json:"status"`
		} `json:"alerts"`
	}{
		Alerts: []struct {
			Labels map[string]string `json:"labels"`
			Status string            `json:"status"`
		}{
			{
				Labels: map[string]string{"pool": "worker-pool", "alertname": "HighLoad"},
				Status: "firing",
			},
			{
				Labels: map[string]string{"pool": "other-pool", "alertname": "LowMem"},
				Status: "resolved", // Should be ignored
			},
		},
	}
	body, _ := json.Marshal(alertPayload)

	req := httptest.NewRequest("POST", "/webhook/worker-pool/prometheus", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", w.Code, http.StatusOK)
	}

	select {
	case intent := <-intentChan:
		if intent.TargetPool != "worker-pool" {
			t.Errorf("Wrong pool: got %s want worker-pool", intent.TargetPool)
		}
		if intent.Action != webhooks.ActionDelta {
			t.Errorf("Wrong action: got %s want delta", intent.Action)
		}
		if intent.Source != "prometheus" {
			t.Errorf("Wrong source: got %s want prometheus", intent.Source)
		}
	default:
		t.Error("No intent received for firing alert")
	}

	// Ensure no second intent (resolved alert ignored)
	select {
	case <-intentChan:
		t.Error("Should ignore resolved alerts")
	default:
		// OK
	}
}
