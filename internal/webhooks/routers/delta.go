package routers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rshade/pulumi-scale/internal/webhooks"
)

// DeltaHandler handles incremental scaling requests.
func DeltaHandler(intentChan chan<- webhooks.ScalingIntent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := chi.URLParam(r, "pool")
		dryRun := r.URL.Query().Get("dryRun") == "true"
		
		var req struct {
			Delta int `json:"delta"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Delta == 0 {
			http.Error(w, "Delta cannot be zero", http.StatusBadRequest)
			return
		}

		intent := webhooks.ScalingIntent{
			TargetPool: pool,
			Action:     webhooks.ActionDelta,
			Value:      req.Delta,
			Source:     "api_delta",
			Reason:     "Manual Delta Request",
			DryRun:     dryRun,
		}

		intentChan <- intent
		w.WriteHeader(http.StatusOK)
	}
}
