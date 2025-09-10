package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

func main() {
	proxyHandler, err := getProxy()
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/inference/", proxyHandler)

	fs := http.FileServer(http.Dir("/var/www/html/pyla"))
	http.Handle("/", fs)

	log.Print("Listening on :3000...")
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatal(err)
	}
}

func getProxy() (http.Handler, error) {
	githubPAT := os.Getenv("GITHUB_MODELS_PAT")
	if githubPAT == "" {
		return nil, errors.New("GITHUB_MODELS_PAT environment variable not set")
	}

	githubModelsURL := os.Getenv("GITHUB_MODELS_URL")
	if githubModelsURL == "" {
		return nil, errors.New("GITHUB_MODELS_URL environment variable not set")
	}

	remote, err := url.Parse(githubModelsURL)
	if err != nil {
		return nil, fmt.Errorf("Error parsing given URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", githubPAT))
		req.Host = remote.Host
	}

	return http.HandlerFunc(proxy.ServeHTTP), nil
}
