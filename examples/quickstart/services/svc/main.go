// A deliberately trivial service. The services are not the point of this
// example; the four planes around them are.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	name := envOr("SERVICE_NAME", "svc")
	version := envOr("SERVICE_VERSION", "0.0.0")
	port := envOr("PORT", "8080")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"service": name,
			"version": version,
		})
	})

	log.Printf("%s %s listening on :%s", name, version, port)
	srv := &http.Server{Addr: ":" + port, Handler: mux}
	log.Fatal(srv.ListenAndServe())
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
