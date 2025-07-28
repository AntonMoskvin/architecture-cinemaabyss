// src/microservices/proxy/main.go
package main

import (
    "fmt"
    "io"
    "log"
    "math/rand"
    "net/http"
    "net/http/httputil"
    "net/url"
    "os"
)

var (
    monolithURL   *url.URL
    moviesServiceURL *url.URL
    migrationPercent int
)

func init() {
    monolithAddr := getEnv("MONOLITH_URL", "http://monolith:8000")
    moviesAddr := getEnv("MOVIES_SERVICE_URL", "http://movies:8001")
    percent := getEnv("MOVIES_MIGRATION_PERCENT", "0")

    monolithURL = parseURL(monolithAddr)
    moviesServiceURL = parseURL(moviesAddr)
    fmt.Sscanf(percent, "%d", &migrationPercent)
}

func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}

func parseURL(rawURL string) *url.URL {
    u, err := url.Parse(rawURL)
    if err != nil {
        log.Fatal("Invalid URL:", rawURL, err)
    }
    return u
}

func main() {
    port := getEnv("PORT", "8000")

    http.HandleFunc("/api/movies", moviesHandler)
    // Все остальные /api/* идут в монолит
    http.HandleFunc("/api/", reverseProxyHandler(monolithURL))

    log.Printf("Proxy запущен на :%s, миграция фильмов: %d%%", port, migrationPercent)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func moviesHandler(w http.ResponseWriter, r *http.Request) {
    // Решаем: куда направить — в монолит или в movies-сервис?
    if shouldRouteToMoviesService() {
        proxy := httputil.NewSingleHostReverseProxy(moviesServiceURL)
        log.Printf("Проксируем /api/movies -> movies-service (%s)", moviesServiceURL)
        proxy.ServeHTTP(w, r)
    } else {
        proxy := httputil.NewSingleHostReverseProxy(monolithURL)
        log.Printf("Проксируем /api/movies -> монолит (%s)", monolithURL)
        proxy.ServeHTTP(w, r)
    }
}

func reverseProxyHandler(target *url.URL) http.HandlerFunc {
    proxy := httputil.NewSingleHostReverseProxy(target)
    return func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Прокси: %s -> %s", r.URL.Path, target)
        proxy.ServeHTTP(w, r)
    }
}

// shouldRouteToMoviesService — решает, направить ли запрос в новый сервис
func shouldRouteToMoviesService() bool {
    return rand.Intn(100) < migrationPercent
}