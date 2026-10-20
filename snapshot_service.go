package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

type objectWriter interface {
	CreateBucket(context.Context, string) error
	PutObject(context.Context, string, string, []byte, string) error
}

type SnapshotService struct {
	store  objectWriter
	bucket string
	now    func() time.Time
}

func (s SnapshotService) Run(ctx context.Context, tenants []Tenant) ([]SnapshotPlan, error) {
	if err := s.store.CreateBucket(ctx, s.bucket); err != nil {
		return nil, fmt.Errorf("prepare snapshot bucket: %w", err)
	}
	plans, err := PlanSnapshots(tenants, s.now())
	if err != nil {
		return nil, err
	}
	for _, plan := range plans {
		idempotencyKey := "snapshot:" + plan.TenantID + ":" + plan.Key
		if err := s.store.PutObject(ctx, s.bucket, plan.Key, plan.Payload, idempotencyKey); err != nil {
			return nil, fmt.Errorf("write %s: %w", plan.Key, err)
		}
	}
	return plans, nil
}

func (s SnapshotService) RunNightly(ctx context.Context, tenants func() []Tenant, hourUTC int) error {
	for {
		now := s.now().UTC()
		next := time.Date(now.Year(), now.Month(), now.Day(), hourUTC, 0, 0, 0, time.UTC)
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		timer := time.NewTimer(next.Sub(now))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			plans, err := s.Run(ctx, tenants())
			if err != nil {
				log.Printf("snapshot run: %v", err)
				continue
			}
			log.Printf("snapshot run complete: objects=%d", len(plans))
		}
	}
}
