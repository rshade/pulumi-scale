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

func TestCloudWatchHandler(t *testing.T) {
	intentChan := make(chan webhooks.ScalingIntent, 1)
	handler := CloudWatchHandler(intentChan)

	r := chi.NewRouter()
	r.Post("/webhook/{pool}/cloudwatch", handler)

	// Test case: Valid Alarm
	alarmPayload := struct {
		NewStateValue string `json:"NewStateValue"`
		AlarmName     string `json:"AlarmName"`
	}{
		NewStateValue: "ALARM",
		AlarmName:     "HighCPU",
	}
	alarmBytes, _ := json.Marshal(alarmPayload)

	snsPayload := struct {
		Type    string `json:"Type"`
		Message string `json:"Message"`
	}{
		Type:    "Notification",
		Message: string(alarmBytes),
	}
	body, _ := json.Marshal(snsPayload)

	req := httptest.NewRequest("POST", "/webhook/worker-pool/cloudwatch", bytes.NewBuffer(body))
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
		if intent.Value != 1 { // Default CloudWatch delta
			t.Errorf("Wrong value: got %d want 1", intent.Value)
		}
		if intent.Source != "cloudwatch" {
			t.Errorf("Wrong source: got %s want cloudwatch", intent.Source)
		}
	default:
		t.Error("No intent received")
	}

	// Test case: SubscriptionConfirmation
	subPayload := struct {
		Type    string `json:"Type"`
		Message string `json:"Message"`
	}{
		Type:    "SubscriptionConfirmation",
		Message: "Confirm me",
	}
	subBody, _ := json.Marshal(subPayload)
	reqSub := httptest.NewRequest("POST", "/webhook/worker-pool/cloudwatch", bytes.NewBuffer(subBody))
	wSub := httptest.NewRecorder()

	r.ServeHTTP(wSub, reqSub)
	if wSub.Code != http.StatusOK {
		t.Errorf("SubscriptionConfirmation returned wrong status: %v", wSub.Code)
	}
	
	// Should NOT receive intent
	select {
	case <-intentChan:
		t.Error("Should not receive intent for SubscriptionConfirmation")
	default:
		// OK
	}
}
