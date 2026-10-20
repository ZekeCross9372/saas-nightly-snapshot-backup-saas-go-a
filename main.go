package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	input := flag.String("input", "tenants.json", "tenant export to snapshot")
	runOnce := flag.Bool("once", false, "run one snapshot immediately")
	flag.Parse()

	client, err := NewStorageClient(os.Getenv("INFRAI_API_KEY"))
	if err != nil {
		fatal(err)
	}
	bucket := envOr("SNAPSHOT_BUCKET", "saas-nightly-snapshots")
	service := SnapshotService{store: client, bucket: bucket, now: time.Now}
	load := func() []Tenant {
		tenants, err := loadTenants(*input)
		if err != nil {
			fatal(err)
		}
		return tenants
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if *runOnce {
		plans, err := service.Run(ctx, load())
		if err != nil {
			fatal(err)
		}
		fmt.Printf("snapshot complete: bucket=%s objects=%d\n", bucket, len(plans))
		return
	}

	hour, err := strconv.Atoi(envOr("SNAPSHOT_HOUR_UTC", "2"))
	if err != nil || hour < 0 || hour > 23 {
		fatal(fmt.Errorf("SNAPSHOT_HOUR_UTC must be from 0 through 23"))
	}
	if err := service.RunNightly(ctx, load, hour); err != nil && ctx.Err() == nil {
		fatal(err)
	}
}

func loadTenants(path string) ([]Tenant, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tenant export: %w", err)
	}
	var tenants []Tenant
	if err := json.Unmarshal(data, &tenants); err != nil {
		return nil, fmt.Errorf("decode tenant export: %w", err)
	}
	return tenants, nil
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
