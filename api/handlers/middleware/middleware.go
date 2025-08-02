package middleware

import (
	"net/http"
	"os"
	"strings"

	"emailchecker/pkg/log"
)

func HostValidation(next http.Handler) http.Handler {
	hosts := os.Getenv("ALLOWED_HOSTS")
	if hosts == "" {
		hosts = "localhost:8080"
	}

	splitted := strings.Split(hosts, ",")
	allowedHosts := make(map[string]bool)
	for _, host := range splitted {
		allowedHosts[host] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host == "" || !allowedHosts[r.Host] {
			log.Set(r.Context(), "host", r.Host)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
