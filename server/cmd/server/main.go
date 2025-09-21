package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	c, err := newCfg()
	if err != nil {
		log.Fatal(err)
	}

	proxyHandler, err := getProxy(c)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.ServeMux{}
	mux.Handle("/inference/", proxyHandler)
	mux.Handle("/", http.FileServer(http.Dir("/var/www/html/pyla")))

	srv := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", c.port),
		Handler: &mux,
	}

	go srv.ListenAndServe()
	log.Printf("Server now listening on port %d", c.port)

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

func getProxy(c *cfg) (http.Handler, error) {
	proxy := httputil.NewSingleHostReverseProxy(c.inferenceUrl)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if strings.HasSuffix(req.Header.Get("Authorization"), c.fairUseToken) {
			log.Print("Proxying inference request using fair use token")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
		}
		req.Host = c.inferenceUrl.Host
	}

	return http.HandlerFunc(proxy.ServeHTTP), nil
}

type cfg struct {
	accessToken  string
	inferenceUrl *url.URL
	fairUseToken string
	port         int
}

func newCfg() (*cfg, error) {
	ghModelsPAT := os.Getenv("GITHUB_MODELS_PAT")
	if ghModelsPAT == "" {
		return nil, errors.New("GITHUB_MODELS_PAT environment variable not set")
	}

	ghModelsURL := os.Getenv("GITHUB_MODELS_URL")
	if ghModelsURL == "" {
		return nil, errors.New("GITHUB_MODELS_URL environment variable not set")
	}
	asURL, err := url.Parse(ghModelsURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse given URL string: %w", err)
	}

	fairUseToken := os.Getenv("FAIR_USE_TOKEN")
	if fairUseToken == "" {
		return nil, errors.New("FAIR_USE_TOKEN environment variable not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		return nil, errors.New("PORT environment variable not set")
	}
	asInt, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse given port value to int: %w", err)
	}

	return &cfg{
		accessToken:  ghModelsPAT,
		inferenceUrl: asURL,
		fairUseToken: fairUseToken,
		port:         asInt,
	}, nil
}
