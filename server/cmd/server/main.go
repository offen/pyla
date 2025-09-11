package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	ghModelsPAT := os.Getenv("GITHUB_MODELS_PAT")
	if ghModelsPAT == "" {
		log.Fatal("GITHUB_MODELS_PAT environment variable not set")
	}

	ghModelsURL := os.Getenv("GITHUB_MODELS_URL")
	if ghModelsURL == "" {
		log.Fatal("GITHUB_MODELS_URL environment variable not set")
	}

	fairUseToken := os.Getenv("FAIR_USE_TOKEN")
	if fairUseToken == "" {
		log.Fatal("FAIR_USE_TOKEN environment variable not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable not set")
	}

	proxyHandler, err := getProxy(ghModelsPAT, ghModelsURL, fairUseToken)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.ServeMux{}
	mux.Handle("/inference/", proxyHandler)
	mux.Handle("/", http.FileServer(http.Dir("/var/www/html/pyla")))

	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%s", port),
		Handler: &mux,
	}

	go srv.ListenAndServe()
	log.Printf("Server now listening on port %s", port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}

	log.Print("Gracefully shut down server")
}

func getProxy(ghModelsPAT, ghModelsURL, fairUseToken string) (http.Handler, error) {
	remote, err := url.Parse(ghModelsURL)
	if err != nil {
		return nil, fmt.Errorf("Error parsing given URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if strings.HasSuffix(req.Header.Get("Authorization"), fairUseToken) {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ghModelsPAT))
		}
		req.Host = remote.Host
	}

	return http.HandlerFunc(proxy.ServeHTTP), nil
}
