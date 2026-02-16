package routers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

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
		defer func() { _ = r.Body.Close() }()

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
		type AlarmMessage struct {
			NewStateValue    string `json:"NewStateValue"` // ALARM, OK, INSUFFICIENT_DATA
			AlarmName        string `json:"AlarmName"`
			AlarmDescription string `json:"AlarmDescription"`
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
			http.Error(w, "Invalid CloudWatch alarm message", http.StatusBadRequest)
			return
		}

		// Determine Scaling Action and Value
		// Priority: Query Params > Alarm Description > Default

		var action webhooks.IntentAction
		var value int
		var configFound bool

		// 1. Check Query Parameters
		q := r.URL.Query()
		actionParam := q.Get("action")
		valueParam := q.Get("value")

		if actionParam != "" && valueParam != "" {
			v, err := strconv.Atoi(valueParam)
			if err == nil {
				// Validate action
				switch webhooks.IntentAction(actionParam) {
				case webhooks.ActionDelta, webhooks.ActionSet:
					action = webhooks.IntentAction(actionParam)
					value = v
					configFound = true
				}
			}
		}

		// 2. Check Alarm Description if not found in query params
		if !configFound && alarmMsg.AlarmDescription != "" {
			// Expected format: {"pulumiscale": {"action": "delta", "value": -1}}
			type PulumiScaleConfig struct {
				Action string `json:"action"`
				Value  int    `json:"value"`
			}
			type DescriptionPayload struct {
				PulumiScale PulumiScaleConfig `json:"pulumiscale"`
			}

			var descPayload DescriptionPayload
			// Try to find the JSON object. It might be mixed with text.
			// For simplicity, attempt to unmarshal the whole string or look for the JSON substring.
			// If the description IS just the JSON, this works.
			if err := json.Unmarshal([]byte(alarmMsg.AlarmDescription), &descPayload); err == nil {
				if descPayload.PulumiScale.Action != "" {
					switch webhooks.IntentAction(descPayload.PulumiScale.Action) {
					case webhooks.ActionDelta, webhooks.ActionSet:
						action = webhooks.IntentAction(descPayload.PulumiScale.Action)
						value = descPayload.PulumiScale.Value
						configFound = true
					}
				}
			}
		}

		// 3. Default Fallback
		if !configFound {
			action = webhooks.ActionDelta
			value = 1
			// Log warning or info about default behavior
			fmt.Printf("CloudWatchHandler: Using default scaling action (Delta +1) for pool %s\n", pool)
		}

		intent := webhooks.ScalingIntent{
			TargetPool: pool,
			Action:     action,
			Value:      value,
			Source:     "cloudwatch",
			Reason:     fmt.Sprintf("CloudWatch Alarm: %s", alarmMsg.AlarmName),
		}

		intentChan <- intent
		w.WriteHeader(http.StatusOK)
	}
}
