package routers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rshade/pulumi-scale/internal/webhooks"
)

// CountHandler handles absolute scaling requests.
func CountHandler(intentChan chan<- webhooks.ScalingIntent) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := chi.URLParam(r, "pool")
		dryRun := r.URL.Query().Get("dryRun") == "true"
		
		var req struct {
			Value int `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Value < 0 {
			http.Error(w, "Value must be non-negative", http.StatusBadRequest)
			return
		}

		intent := webhooks.ScalingIntent{
			TargetPool: pool,
			Action:     webhooks.ActionSet,
			Value:      req.Value,
			Source:     "api_count",
			Reason:     "Manual Set Request",
			DryRun:     dryRun,
		}

		intentChan <- intent
		w.WriteHeader(http.StatusOK)
	}
}
