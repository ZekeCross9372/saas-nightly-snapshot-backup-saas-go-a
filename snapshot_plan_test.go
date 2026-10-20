package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPlanSnapshotsAppliesLifecyclePolicy(t *testing.T) {
	asOf := time.Date(2026, 9, 6, 2, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		tenants   []Tenant
		wantPlans int
		wantRows  int
		wantKey   string
		wantFirst string
	}{
		{
			name: "active tenant includes only active accounts in stable order",
			tenants: []Tenant{{ID: "north", Name: "Northwind", Status: "active", Accounts: []Account{
				{ID: "b", Name: "Beta", Lifecycle: "active"},
				{ID: "c", Name: "Closed", Lifecycle: "closed"},
				{ID: "a", Name: "Alpha", Lifecycle: "active"},
			}}},
			wantPlans: 1, wantRows: 2,
			wantKey: "tenant=north/date=2026-09-06/snapshot.json", wantFirst: "a",
		},
		{
			name: "paused tenant emits no object",
			tenants: []Tenant{{ID: "south", Name: "Southwind", Status: "paused", Accounts: []Account{
				{ID: "a", Name: "Alpha", Lifecycle: "active"},
			}}},
			wantPlans: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plans, err := PlanSnapshots(tt.tenants, asOf)
			if err != nil {
				t.Fatal(err)
			}
			if len(plans) != tt.wantPlans {
				t.Fatalf("plans=%d, want %d", len(plans), tt.wantPlans)
			}
			if tt.wantPlans == 0 {
				return
			}
			if plans[0].Key != tt.wantKey {
				t.Fatalf("key=%q, want %q", plans[0].Key, tt.wantKey)
			}
			var snapshot Snapshot
			if err := json.Unmarshal(plans[0].Payload, &snapshot); err != nil {
				t.Fatal(err)
			}
			if snapshot.RowCount != tt.wantRows || snapshot.Accounts[0].ID != tt.wantFirst {
				t.Fatalf("snapshot rows=%d first=%q", snapshot.RowCount, snapshot.Accounts[0].ID)
			}
		})
	}
}
