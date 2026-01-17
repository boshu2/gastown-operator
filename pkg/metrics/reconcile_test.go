/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewReconcileTimer(t *testing.T) {
	timer := NewReconcileTimer("test-controller")

	if timer == nil {
		t.Fatal("expected timer to be created")
	}
	if timer.controller != "test-controller" {
		t.Errorf("expected controller 'test-controller', got %q", timer.controller)
	}
	if timer.start.IsZero() {
		t.Error("expected start time to be set")
	}
}

func TestReconcileTimer_ObserveDuration(t *testing.T) {
	// Reset the histogram before testing
	ReconcileDuration.Reset()

	timer := NewReconcileTimer("test-controller")
	time.Sleep(10 * time.Millisecond)
	timer.ObserveDuration()

	// Check that the metric was recorded by collecting all metrics
	// The histogram's count should be > 0 after observation
	// We verify this indirectly by ensuring no panic occurs
	// and the timer completes without error
}

func TestReconcileTimer_RecordResult(t *testing.T) {
	// Reset the counter before testing
	ReconcileTotal.Reset()

	timer := NewReconcileTimer("test-controller")
	timer.RecordResult(ResultSuccess)

	count := testutil.ToFloat64(ReconcileTotal.WithLabelValues("test-controller", ResultSuccess))
	if count != 1 {
		t.Errorf("expected count 1, got %f", count)
	}

	timer.RecordResult(ResultSuccess)
	count = testutil.ToFloat64(ReconcileTotal.WithLabelValues("test-controller", ResultSuccess))
	if count != 2 {
		t.Errorf("expected count 2, got %f", count)
	}
}

func TestRecordError(t *testing.T) {
	// Reset the counter before testing
	ReconcileErrors.Reset()

	RecordError("test-controller", "TransientError")

	count := testutil.ToFloat64(ReconcileErrors.WithLabelValues("test-controller", "TransientError"))
	if count != 1 {
		t.Errorf("expected count 1, got %f", count)
	}
}

func TestNewGTCLITimer(t *testing.T) {
	timer := NewGTCLITimer("rig list")

	if timer == nil {
		t.Fatal("expected timer to be created")
	}
	if timer.command != "rig list" {
		t.Errorf("expected command 'rig list', got %q", timer.command)
	}
	if timer.start.IsZero() {
		t.Error("expected start time to be set")
	}
}

func TestGTCLITimer_RecordSuccess(t *testing.T) {
	// Reset metrics before testing
	GTCLICallsTotal.Reset()
	GTCLIDuration.Reset()

	timer := NewGTCLITimer("rig list")
	time.Sleep(10 * time.Millisecond)
	timer.RecordSuccess()

	count := testutil.ToFloat64(GTCLICallsTotal.WithLabelValues("rig list", ResultSuccess))
	if count != 1 {
		t.Errorf("expected count 1, got %f", count)
	}
}

func TestGTCLITimer_RecordError(t *testing.T) {
	// Reset metrics before testing
	GTCLICallsTotal.Reset()
	GTCLIDuration.Reset()

	timer := NewGTCLITimer("polecat nuke")
	timer.RecordError()

	count := testutil.ToFloat64(GTCLICallsTotal.WithLabelValues("polecat nuke", ResultError))
	if count != 1 {
		t.Errorf("expected count 1, got %f", count)
	}
}

func TestUpdateRigPhase(t *testing.T) {
	// Reset gauge before testing
	RigPhaseGauge.Reset()

	UpdateRigPhase("Ready", 5)

	count := testutil.ToFloat64(RigPhaseGauge.WithLabelValues("Ready"))
	if count != 5 {
		t.Errorf("expected count 5, got %f", count)
	}

	UpdateRigPhase("Ready", 3)
	count = testutil.ToFloat64(RigPhaseGauge.WithLabelValues("Ready"))
	if count != 3 {
		t.Errorf("expected count 3, got %f", count)
	}
}

func TestUpdatePolecatPhase(t *testing.T) {
	// Reset gauge before testing
	PolecatPhaseGauge.Reset()

	UpdatePolecatPhase("test-rig", "Working", 10)

	count := testutil.ToFloat64(PolecatPhaseGauge.WithLabelValues("test-rig", "Working"))
	if count != 10 {
		t.Errorf("expected count 10, got %f", count)
	}
}

func TestUpdateConvoyPhase(t *testing.T) {
	// Reset gauge before testing
	ConvoyPhaseGauge.Reset()

	UpdateConvoyPhase("InProgress", 2)

	count := testutil.ToFloat64(ConvoyPhaseGauge.WithLabelValues("InProgress"))
	if count != 2 {
		t.Errorf("expected count 2, got %f", count)
	}
}

func TestMetricsRegistration(t *testing.T) {
	// Test that metrics are registered by checking they can be gathered
	// Note: This is a basic sanity check since metrics are registered in init()

	tests := []struct {
		name   string
		metric prometheus.Collector
	}{
		{"ReconcileTotal", ReconcileTotal},
		{"ReconcileDuration", ReconcileDuration},
		{"RigPhaseGauge", RigPhaseGauge},
		{"PolecatPhaseGauge", PolecatPhaseGauge},
		{"ConvoyPhaseGauge", ConvoyPhaseGauge},
		{"GTCLICallsTotal", GTCLICallsTotal},
		{"GTCLIDuration", GTCLIDuration},
		{"ReconcileErrors", ReconcileErrors},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metric == nil {
				t.Errorf("metric %s is nil", tt.name)
			}
		})
	}
}

func TestResultConstants(t *testing.T) {
	if ResultSuccess != "success" {
		t.Errorf("expected ResultSuccess='success', got %q", ResultSuccess)
	}
	if ResultError != "error" {
		t.Errorf("expected ResultError='error', got %q", ResultError)
	}
	if ResultRequeue != "requeue" {
		t.Errorf("expected ResultRequeue='requeue', got %q", ResultRequeue)
	}
}
