// Command worker is the background job/cron process. Step 0 ships a stub
// that logs a heartbeat and exits; scheduled jobs and queue consumers land
// once modules need them.
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XoDeR/empops/api-go/pkg/logger"
)

func main() {
	log := logger.New(envOrDefault("EMPOPS_LOG_LEVEL", "info"))
	log.Info("worker started")
	fmt.Println("worker started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case sig := <-quit:
			log.Info("worker stopping", "signal", sig.String())
			return
		case <-ticker.C:
			log.Info("worker heartbeat: no scheduled jobs yet (Step 0 stub)")
		}
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
