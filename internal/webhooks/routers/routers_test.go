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

func TestCountHandler(t *testing.T) {
	intentChan := make(chan webhooks.ScalingIntent, 1)
	handler := CountHandler(intentChan)

	// Setup Router to handle URL params
	r := chi.NewRouter()
	r.Post("/webhook/{pool}/count", handler)

	payload := map[string]int{"value": 10}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhook/worker-pool/count", bytes.NewBuffer(body))
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
		if intent.Value != 10 {
			t.Errorf("Wrong value: got %d want 10", intent.Value)
		}
		if intent.Action != webhooks.ActionSet {
			t.Errorf("Wrong action: got %s want set", intent.Action)
		}
	default:
		t.Error("No intent received")
	}
}

func TestDeltaHandler(t *testing.T) {
	intentChan := make(chan webhooks.ScalingIntent, 1)
	handler := DeltaHandler(intentChan)

	r := chi.NewRouter()
	r.Post("/webhook/{pool}/delta", handler)

	payload := map[string]int{"delta": -1}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/webhook/worker-pool/delta", bytes.NewBuffer(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", w.Code, http.StatusOK)
	}

	select {
	case intent := <-intentChan:
		if intent.Value != -1 {
			t.Errorf("Wrong value: got %d want -1", intent.Value)
		}
		if intent.Action != webhooks.ActionDelta {
			t.Errorf("Wrong action: got %s want delta", intent.Action)
		}
	default:
		t.Error("No intent received")
	}
}
