package handler

import (
	"net/http"
)

func HealthCheck() error {
	http.HandleFunc("/health_check", ok)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return err
	}
	return nil
}

func ok(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
