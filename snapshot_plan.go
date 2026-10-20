package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type Account struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Lifecycle string `json:"lifecycle"`
}

type Tenant struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	Accounts []Account `json:"accounts"`
}

type Snapshot struct {
	TenantID  string    `json:"tenant_id"`
	AsOf      time.Time `json:"as_of"`
	Accounts  []Account `json:"accounts"`
	RowCount  int       `json:"row_count"`
	SchemaRev int       `json:"schema_rev"`
}

type SnapshotPlan struct {
	TenantID string
	Key      string
	Payload  []byte
}

// PlanSnapshots is the pipeline boundary: paused tenants and closed accounts
// are omitted before bytes reach object storage.
func PlanSnapshots(tenants []Tenant, asOf time.Time) ([]SnapshotPlan, error) {
	day := asOf.UTC().Format("2006-01-02")
	plans := make([]SnapshotPlan, 0, len(tenants))
	for _, tenant := range tenants {
		if tenant.ID == "" || tenant.Name == "" {
			return nil, fmt.Errorf("tenant id and name are required")
		}
		if tenant.Status != "active" {
			continue
		}

		accounts := make([]Account, 0, len(tenant.Accounts))
		for _, account := range tenant.Accounts {
			if account.Lifecycle == "active" {
				accounts = append(accounts, account)
			}
		}
		sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
		payload, err := json.Marshal(Snapshot{
			TenantID: tenant.ID, AsOf: asOf.UTC(), Accounts: accounts,
			RowCount: len(accounts), SchemaRev: 1,
		})
		if err != nil {
			return nil, fmt.Errorf("encode tenant %s: %w", tenant.ID, err)
		}
		plans = append(plans, SnapshotPlan{
			TenantID: tenant.ID,
			Key:      fmt.Sprintf("tenant=%s/date=%s/snapshot.json", tenant.ID, day),
			Payload:  payload,
		})
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].TenantID < plans[j].TenantID })
	return plans, nil
}
