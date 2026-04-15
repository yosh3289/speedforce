package main

import (
	"context"
	"log"
	"time"

	"github.com/yosh3289/speedforce/internal/core"
	"github.com/yosh3289/speedforce/internal/probe"
	"github.com/yosh3289/speedforce/internal/ui/tray"
)

func main() {
	client := probe.NewClient(probe.ClientOptions{
		Timeout:   5 * time.Second,
		ProxyMode: string(probe.ProxyAuto),
	})
	prober := probe.NewHTTPSProber(client)

	t := tray.New(
		func() { log.Println("Detail requested (window not implemented yet)") },
		func() { log.Println("Quit") },
	)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		tick := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			res := prober.Probe(ctx, "Claude", "https://claude.ai")
			status := core.StatusHealthy
			if !res.IsUp() {
				status = core.StatusDown
			}
			t.SetStatus(status)
			log.Printf("probe: up=%v latency=%dms err=%v", res.IsUp(), res.LatencyMs, res.Err)
		}
		tick()
		for range ticker.C {
			tick()
		}
	}()

	t.Run()
}
