package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
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

	proxyHandler, err := getProxy(ghModelsPAT, ghModelsURL, fairUseToken)
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
