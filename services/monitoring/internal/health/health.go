package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

func Handler(checks map[string]Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		results := make(map[string]string, len(checks))
		status := http.StatusOK
		overall := "ok"

		for name, p := range checks {
			if err := p.Ping(ctx); err != nil {
				results[name] = err.Error()
				status = http.StatusServiceUnavailable
				overall = "unhealthy"
				continue
			}
			results[name] = "ok"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": overall,
			"checks": results,
		})
	}
}
