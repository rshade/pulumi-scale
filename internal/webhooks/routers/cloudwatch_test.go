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
	intentChan := make(chan webhooks.ScalingIntent, 10)
	handler := CloudWatchHandler(intentChan)

	r := chi.NewRouter()
	r.Post("/webhook/{pool}/cloudwatch", handler)

	type AlarmPayload struct {
		NewStateValue    string `json:"NewStateValue"`
		AlarmName        string `json:"AlarmName"`
		AlarmDescription string `json:"AlarmDescription,omitempty"`
	}

	type SNSPayload struct {
		Type    string `json:"Type"`
		Message string `json:"Message"`
	}

	tests := []struct {
		name       string
		url        string
		alarm      AlarmPayload
		snsType    string
		wantAction webhooks.IntentAction
		wantValue  int
		wantIntent bool
	}{
		{
			name: "Default Behavior (Scale Up)",
			url:  "/webhook/worker-pool/cloudwatch",
			alarm: AlarmPayload{
				NewStateValue: "ALARM",
				AlarmName:     "HighCPU",
			},
			snsType:    "Notification",
			wantAction: webhooks.ActionDelta,
			wantValue:  1,
			wantIntent: true,
		},
		{
			name: "Query Params: Scale Down",
			url:  "/webhook/worker-pool/cloudwatch?action=delta&value=-1",
			alarm: AlarmPayload{
				NewStateValue: "ALARM",
				AlarmName:     "LowCPU",
			},
			snsType:    "Notification",
			wantAction: webhooks.ActionDelta,
			wantValue:  -1,
			wantIntent: true,
		},
		{
			name: "Query Params: Set Value",
			url:  "/webhook/worker-pool/cloudwatch?action=set&value=5",
			alarm: AlarmPayload{
				NewStateValue: "ALARM",
				AlarmName:     "SpecificLoad",
			},
			snsType:    "Notification",
			wantAction: webhooks.ActionSet,
			wantValue:  5,
			wantIntent: true,
		},
		{
			name: "Alarm Description: Scale Down",
			url:  "/webhook/worker-pool/cloudwatch",
			alarm: AlarmPayload{
				NewStateValue:    "ALARM",
				AlarmName:        "LowMemory",
				AlarmDescription: `{"pulumiscale": {"action": "delta", "value": -1}}`,
			},
			snsType:    "Notification",
			wantAction: webhooks.ActionDelta,
			wantValue:  -1,
			wantIntent: true,
		},
		{
			name: "Priority: Query Params > Description",
			url:  "/webhook/worker-pool/cloudwatch?action=delta&value=2",
			alarm: AlarmPayload{
				NewStateValue:    "ALARM",
				AlarmName:        "ConflictingConfig",
				AlarmDescription: `{"pulumiscale": {"action": "delta", "value": -5}}`,
			},
			snsType:    "Notification",
			wantAction: webhooks.ActionDelta,
			wantValue:  2,
			wantIntent: true,
		},
		{
			name: "Non-Alarm State",
			url:  "/webhook/worker-pool/cloudwatch",
			alarm: AlarmPayload{
				NewStateValue: "OK",
				AlarmName:     "Recovered",
			},
			snsType:    "Notification",
			wantIntent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alarmBytes, _ := json.Marshal(tt.alarm)
			snsPayload := SNSPayload{
				Type:    tt.snsType,
				Message: string(alarmBytes),
			}
			body, _ := json.Marshal(snsPayload)

			req := httptest.NewRequest("POST", tt.url, bytes.NewBuffer(body))
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Handler returned wrong status code: got %v want %v", w.Code, http.StatusOK)
			}

			if tt.wantIntent {
				select {
				case intent := <-intentChan:
					if intent.Action != tt.wantAction {
						t.Errorf("Wrong action: got %s want %s", intent.Action, tt.wantAction)
					}
					if intent.Value != tt.wantValue {
						t.Errorf("Wrong value: got %d want %d", intent.Value, tt.wantValue)
					}
					if intent.TargetPool != "worker-pool" {
						t.Errorf("Wrong pool: got %s want worker-pool", intent.TargetPool)
					}
				default:
					t.Error("No intent received but expected one")
				}
			} else {
				select {
				case <-intentChan:
					t.Error("Received intent but expected none")
				default:
					// OK
				}
			}
		})
	}

	t.Run("SubscriptionConfirmation", func(t *testing.T) {
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

		select {
		case <-intentChan:
			t.Error("Should not receive intent for SubscriptionConfirmation")
		default:
			// OK
		}
	})

	t.Run("Invalid Alarm Message", func(t *testing.T) {
		snsPayload := SNSPayload{
			Type:    "Notification",
			Message: "not valid json at all",
		}
		body, _ := json.Marshal(snsPayload)
		req := httptest.NewRequest("POST", "/webhook/worker-pool/cloudwatch", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Invalid alarm message returned wrong status: got %v want %v", w.Code, http.StatusBadRequest)
		}

		select {
		case <-intentChan:
			t.Error("Should not receive intent for invalid alarm message")
		default:
			// OK
		}
	})
}
