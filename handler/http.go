package handler

import (
	"context"
	"net/http"
)

func HealthCheck(ctx context.Context) error {
	http.HandleFunc("/health_check", ok)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func ok(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
