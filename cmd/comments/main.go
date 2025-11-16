package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "instant/internal/comments"
    "instant/internal/consul"
    "instant/internal/database"
)

func main() {
    port := getEnv("COMMENTS_SERVICE_PORT", "8085")
    host := getEnv("COMMENTS_SERVICE_HOST", "comments-service")
    consulAddr := getEnv("CONSUL_HTTP_ADDR", "localhost:8500")
    consulToken := getEnv("CONSUL_HTTP_TOKEN", "")

    log.Println("Starting Comments Service...")
    log.Printf("Host: %s Port: %s Consul: %s", host, port, consulAddr)

    db := database.New()
    defer db.Close()

    svc := comments.NewService(db)
    router := comments.SetupRouter(svc)

    cClient, err := consul.NewClientWithToken(consulAddr, consulToken)
    if err != nil {
        log.Fatalf("consul client error: %v", err)
    }

    serviceID := fmt.Sprintf("comments-service-%s", host)
    _ = cClient.Deregister(serviceID)

    if err := cClient.Register(&consul.ServiceConfig{
        ID:      serviceID,
        Name:    "comments-service",
        Address: host,
        Port:    mustAtoi(port),
        Tags:    []string{"comments", "social"},
        Check: &consul.HealthCheck{
            HTTP:     fmt.Sprintf("http://%s:%s/health", host, port),
            Interval: "10s",
            Timeout:  "3s",
        },
    }); err != nil {
        log.Fatalf("consul register: %v", err)
    }

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      router,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go func() {
        log.Printf("Comments Service listening on :%s", port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("listen: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    _ = cClient.Deregister(serviceID)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(ctx)
}

func getEnv(k, def string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return def
}

func mustAtoi(s string) int {
    var n int
    fmt.Sscanf(s, "%d", &n)
    return n
}
