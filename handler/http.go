package handler

import (
	"context"
	"net/http"
)

func HealthCheck(ctx context.Context) error {
	http.HandleFunc("/health_check", ok)

	errCh := make(chan error)
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func ok(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
