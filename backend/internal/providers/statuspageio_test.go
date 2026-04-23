package providers

import (
	"testing"
	"time"
)

func TestSummaryToStatusTrustsIncidentsOverPageRollup(t *testing.T) {
	// Real-world case from 2026-04-23: githubstatus.com reported
	// status.indicator="none" ("All Systems Operational") but carried an
	// unresolved critical incident. The rollup can lag operator action, so
	// the adapter must upgrade the indicator to match the worst active
	// incident impact.
	s := &statuspageSummary{}
	s.Status.Indicator = "none"
	s.Status.Description = "All Systems Operational"
	s.Incidents = append(s.Incidents, struct {
		ID        string     `json:"id"`
		Name      string     `json:"name"`
		Status    string     `json:"status"`
		Impact    string     `json:"impact"`
		Shortlink string     `json:"shortlink"`
		UpdatedAt time.Time  `json:"updated_at"`
		Resolved  *time.Time `json:"resolved_at"`
	}{
		ID:     "abc",
		Name:   "Incident with multiple GitHub services",
		Status: "investigating",
		Impact: "critical",
	})

	out := summaryToStatus(s)

	if out.Indicator != IndicatorCritical {
		t.Errorf("indicator = %s, want %s", out.Indicator, IndicatorCritical)
	}
	if out.Description == "All Systems Operational" {
		t.Errorf("description still reports operational: %q", out.Description)
	}
	if len(out.Incidents) != 1 {
		t.Fatalf("incidents = %d, want 1", len(out.Incidents))
	}
}

func TestSummaryToStatusKeepsPageRollupWhenWorse(t *testing.T) {
	// Inverse: page indicator is worse than any active incident impact
	// (e.g. operators set "major" manually to broadcast a banner while
	// listed incidents are only "minor"). We must not downgrade.
	s := &statuspageSummary{}
	s.Status.Indicator = "major"
	s.Status.Description = "Partial Outage"
	s.Incidents = append(s.Incidents, struct {
		ID        string     `json:"id"`
		Name      string     `json:"name"`
		Status    string     `json:"status"`
		Impact    string     `json:"impact"`
		Shortlink string     `json:"shortlink"`
		UpdatedAt time.Time  `json:"updated_at"`
		Resolved  *time.Time `json:"resolved_at"`
	}{
		ID:     "x",
		Name:   "small thing",
		Status: "investigating",
		Impact: "minor",
	})

	out := summaryToStatus(s)

	if out.Indicator != IndicatorMajor {
		t.Errorf("indicator = %s, want %s", out.Indicator, IndicatorMajor)
	}
	if out.Description != "Partial Outage" {
		t.Errorf("description overwritten: %q", out.Description)
	}
}

func TestSummaryToStatusSkipsResolvedIncidents(t *testing.T) {
	// A resolved incident must not trigger the upgrade path.
	now := time.Now()
	s := &statuspageSummary{}
	s.Status.Indicator = "none"
	s.Status.Description = "All Systems Operational"
	s.Incidents = append(s.Incidents, struct {
		ID        string     `json:"id"`
		Name      string     `json:"name"`
		Status    string     `json:"status"`
		Impact    string     `json:"impact"`
		Shortlink string     `json:"shortlink"`
		UpdatedAt time.Time  `json:"updated_at"`
		Resolved  *time.Time `json:"resolved_at"`
	}{
		ID:       "y",
		Name:     "already done",
		Status:   "resolved",
		Impact:   "critical",
		Resolved: &now,
	})

	out := summaryToStatus(s)

	if out.Indicator != IndicatorOperational {
		t.Errorf("indicator = %s, want operational (resolved incident shouldn't upgrade)", out.Indicator)
	}
	if len(out.Incidents) != 0 {
		t.Errorf("incidents = %d, want 0", len(out.Incidents))
	}
}
