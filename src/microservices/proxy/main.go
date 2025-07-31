package main

import (
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "net/http/httputil"
    "net/url"
    "os"
    "encoding/json"  // Добавлено: для json.NewEncoder и json.Marshal
)

var (
    monolithTarget   *url.URL
    moviesServiceTarget *url.URL
    migrationPercent int
)

func init() {
    var err error
    // Получаем URL из переменных окружения
    monolithAddr := getEnv("MONOLITH_URL", "http://monolith:8080")
    moviesAddr := getEnv("MOVIES_SERVICE_URL", "http://movies-service:8001")
    percentStr := getEnv("MOVIES_MIGRATION_PERCENT", "0")

    monolithTarget, err = url.Parse(monolithAddr)
    if err != nil {
        log.Fatal("Invalid MONOLITH_URL:", err)
    }

    moviesServiceTarget, err = url.Parse(moviesAddr)
    if err != nil {
        log.Fatal("Invalid MOVIES_SERVICE_URL:", err)
    }

    _, err = fmt.Sscanf(percentStr, "%d", &migrationPercent)
    if err != nil {
        migrationPercent = 0
    }
    if migrationPercent < 0 {
        migrationPercent = 0
    }
    if migrationPercent > 100 {
        migrationPercent = 100
    }
}

func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}

func main() {
    port := getEnv("PORT", "8000")

    // Health check
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]bool{"status": true})
    })

    // Прокси для /api/movies — Strangler Fig
    http.HandleFunc("/api/movies", func(w http.ResponseWriter, r *http.Request) {
        if shouldRouteToMoviesService() {
            proxy := httputil.NewSingleHostReverseProxy(moviesServiceTarget)
            log.Printf("[PROXY] /api/movies → movies-service (%s)", moviesServiceTarget)
            proxy.ServeHTTP(w, r)
        } else {
            proxy := httputil.NewSingleHostReverseProxy(monolithTarget)
            log.Printf("[PROXY] /api/movies → monolith (%s)", monolithTarget)
            proxy.ServeHTTP(w, r)
        }
    })

    // Все остальные /api/* → монолит
    http.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
        proxy := httputil.NewSingleHostReverseProxy(monolithTarget)
        log.Printf("[PROXY] %s → monolith", r.URL.Path)
        proxy.ServeHTTP(w, r)
    })

    log.Printf("✅ Proxy запущен на :%s", port)
    log.Printf("➡️  Миграция /api/movies: %d%% в movies-service", migrationPercent)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}

func shouldRouteToMoviesService() bool {
    return rand.Intn(100) < migrationPercent
}