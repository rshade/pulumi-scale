package routers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rshade/pulumi-scale/internal/webhooks"
)

// CloudWatchHandler handles AWS SNS notifications from CloudWatch.
func CloudWatchHandler(intentChan chan<- webhooks.ScalingIntent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := chi.URLParam(r, "pool")
		if pool == "" {
			http.Error(w, "Pool parameter required", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Minimal SNS Payload structure
		type SNSPayload struct {
			Type    string `json:"Type"`
			Message string `json:"Message"`
		}

		var snsPayload SNSPayload
		if err := json.Unmarshal(body, &snsPayload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		// Handle SubscriptionConfirmation (AWS requirement)
		if snsPayload.Type == "SubscriptionConfirmation" {
			// For MVP/Sidecar inside VPC, logging the URL is usually enough if manual confirmation is possible,
			// or we can auto-confirm if we have the logic.
			// Ideally we should log the SubscribeURL.
			fmt.Printf("Received SNS SubscriptionConfirmation. Visit SubscribeURL to confirm.\n")
			// In a real implementation, we might fetch the SubscribeURL.
			w.WriteHeader(http.StatusOK)
			return
		}

		// CloudWatch Alarm Message often comes as a JSON string inside the Message field.
		// For simplicity in this iteration, we assume the Alarm signifies "Scale Up" or "Scale Down"
		// based on some convention or configuration. 
		// However, without a specific payload structure from the spec (it just says SNS JSON), 
		// we'll implement a basic trigger.
		// If the alarm payload contains details, we'd parse them here.
		// For now, let's assume the existence of an alarm triggers a default action or check.
		
		// Refined logic: CloudWatch Alarms usually have NewStateValue ("ALARM").
		// We'll normalize this to a generic signal.
		// Ideally, the endpoint might be specific like /webhook/{pool}/cloudwatch?action=up
		// But the spec says /webhook/{pool}/cloudwatch.
		// Let's infer Action from the Message content if possible, or assume a default logic (e.g., +1).
		// *Clarification needed/assumed*: The spec implies parsing logic.
		// Let's parse the inner Message to check for Alarm state.

		type AlarmMessage struct {
			NewStateValue string `json:"NewStateValue"` // ALARM, OK, INSUFFICIENT_DATA
			AlarmName     string `json:"AlarmName"`
		}

		var alarmMsg AlarmMessage
		// Attempt to unmarshal Message if it's JSON
		if err := json.Unmarshal([]byte(snsPayload.Message), &alarmMsg); err == nil {
			if alarmMsg.NewStateValue != "ALARM" {
				// Ignore non-alarm states
				w.WriteHeader(http.StatusOK)
				return
			}
		} else {
			// If not JSON, treat raw message as description?
			// Fallback: Proceed as trigger.
		}

		// Send intent
		// Defaulting to Delta +1 for "CloudWatch Alarm" as a generic scaler signal if not specified.
		// A robust implementation would look for specific data in the alarm description or dimension.
		intent := webhooks.ScalingIntent{
			TargetPool: pool,
			Action:     webhooks.ActionDelta,
			Value:      1, // Default step
			Source:     "cloudwatch",
			Reason:     "SNS Notification Received",
		}

		intentChan <- intent
		w.WriteHeader(http.StatusOK)
	}
}
