package main

import (
	"context"
	"fmt"
	"github.com/cockroachdb/errors"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client, baseURL, err := newGoogleClient(ctx)
	if err != nil {
		log.Printf("[ERROR] error newGoogleClient: %s", err)
		cancel()
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		promURL := fmt.Sprintf("%s%s?%s", baseURL, r.URL.Path, r.URL.RawQuery)
		defer r.Body.Close()
		req, err := http.NewRequestWithContext(ctx, r.Method, promURL, r.Body)
		if err != nil {
			handle500(w, errors.Wrap(err, "error http.NewRequestWithContext"))
			return
		}
		req.Header = r.Header
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			handle500(w, errors.Wrap(err, "error client.Do"))
			return
		}
		log.Printf("[INFO] request Header: %s", resp.Request.Header)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			handle500(w, errors.Wrap(err, "error io.ReadAll"))
			return
		}
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		log.Printf("[INFO] %s %s %s", r.Method, promURL, resp.Status)
		log.Printf("[INFO] %s", string(body))
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	go func() {
		log.Printf("[INFO] Listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ERROR] error ListenAndServe: %#v", err)
			cancel()
		}
	}()
	<-ctx.Done()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] error Shutdown: %#v", err)
	}
}

func newGoogleClient(ctx context.Context) (*http.Client, string, error) {
	client, _, err := transport.NewHTTPClient(ctx,
		option.WithEndpoint("https://monitoring.googleapis.com"),
		option.WithScopes(
			"https://www.googleapis.com/auth/cloud-platform",
		))
	if err != nil {
		return nil, "", errors.Wrap(err, "error transport.NewHTTPClient")
	}
	project := os.Getenv("GOOGLE_PROJECT_ID")
	if project == "" {
		return nil, "", errors.New("GOOGLE_PROJECT_ID must be set")
	}
	const location = "global" // See https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.location.prometheus.api.v1/query
	return client, fmt.Sprintf("/v1/projects/%s/locations/%s/prometheus", project, location), nil
}

func handle500(w http.ResponseWriter, err error) {
	log.Printf("[ERROR] 500 Internal Server Error: %s", err)
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}
