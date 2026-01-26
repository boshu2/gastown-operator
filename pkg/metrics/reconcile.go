// Package metrics provides Prometheus metrics for the Gas Town operator.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	// Labels for metrics
	labelController = "controller"
	labelResult     = "result"
	labelRig        = "rig"
	labelPhase      = "phase"

	// Result values
	ResultSuccess = "success"
	ResultError   = "error"
	ResultRequeue = "requeue"
)

var (
	// ReconcileTotal counts the total number of reconciliations per controller and result.
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gastown_reconcile_total",
			Help: "Total number of reconciliations by controller and result",
		},
		[]string{labelController, labelResult},
	)

	// ReconcileDuration tracks the duration of reconciliation loops.
	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gastown_reconcile_duration_seconds",
			Help:    "Duration of reconciliation loops in seconds",
			Buckets: []float64{0.001, 0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{labelController},
	)

	// RigPhaseGauge tracks the number of rigs in each phase.
	RigPhaseGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gastown_rig_phase_total",
			Help: "Number of rigs in each phase",
		},
		[]string{labelPhase},
	)

	// PolecatPhaseGauge tracks the number of polecats in each phase per rig.
	PolecatPhaseGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gastown_polecat_phase_total",
			Help: "Number of polecats in each phase by rig",
		},
		[]string{labelRig, labelPhase},
	)

	// ConvoyPhaseGauge tracks the number of convoys in each phase.
	ConvoyPhaseGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gastown_convoy_phase_total",
			Help: "Number of convoys in each phase",
		},
		[]string{labelPhase},
	)

	// GTCLICallsTotal tracks gt CLI invocations.
	GTCLICallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gastown_gt_cli_calls_total",
			Help: "Total number of gt CLI calls by command and result",
		},
		[]string{"command", labelResult},
	)

	// GTCLIDuration tracks gt CLI call durations.
	GTCLIDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gastown_gt_cli_duration_seconds",
			Help:    "Duration of gt CLI calls in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		},
		[]string{"command"},
	)

	// ReconcileErrors tracks errors during reconciliation with more detail.
	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gastown_reconcile_errors_total",
			Help: "Total number of reconciliation errors by controller and error type",
		},
		[]string{labelController, "error_type"},
	)

	// RefineryMergeTotal counts merge attempts by rig and result.
	RefineryMergeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gastown_refinery_merge_total",
			Help: "Total number of merge attempts by rig and result",
		},
		[]string{labelRig, labelResult},
	)

	// RefineryMergeDuration tracks the duration of merge operations.
	RefineryMergeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gastown_refinery_merge_duration_seconds",
			Help:    "Duration of merge operations in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
		},
		[]string{labelRig},
	)

	// RefineryConflictsTotal counts merge conflicts by rig.
	RefineryConflictsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gastown_refinery_conflicts_total",
			Help: "Total number of merge conflicts by rig",
		},
		[]string{labelRig},
	)

	// RefineryQueueLength tracks the number of items in the merge queue per rig.
	RefineryQueueLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gastown_refinery_queue_length",
			Help: "Number of items in the merge queue by rig",
		},
		[]string{labelRig},
	)
)

func init() {
	// Register metrics with controller-runtime's metrics registry
	metrics.Registry.MustRegister(
		ReconcileTotal,
		ReconcileDuration,
		RigPhaseGauge,
		PolecatPhaseGauge,
		ConvoyPhaseGauge,
		GTCLICallsTotal,
		GTCLIDuration,
		ReconcileErrors,
		RefineryMergeTotal,
		RefineryMergeDuration,
		RefineryConflictsTotal,
		RefineryQueueLength,
	)
}

// ReconcileTimer is a helper to time reconciliation loops.
type ReconcileTimer struct {
	controller string
	start      time.Time
}

// NewReconcileTimer creates a new timer for a reconciliation.
func NewReconcileTimer(controller string) *ReconcileTimer {
	return &ReconcileTimer{
		controller: controller,
		start:      time.Now(),
	}
}

// ObserveDuration records the reconciliation duration.
func (t *ReconcileTimer) ObserveDuration() {
	duration := time.Since(t.start).Seconds()
	ReconcileDuration.WithLabelValues(t.controller).Observe(duration)
}

// RecordResult records the result of a reconciliation.
func (t *ReconcileTimer) RecordResult(result string) {
	ReconcileTotal.WithLabelValues(t.controller, result).Inc()
}

// RecordError records an error with its type.
func RecordError(controller, errorType string) {
	ReconcileErrors.WithLabelValues(controller, errorType).Inc()
}

// GTCLITimer tracks gt CLI call timing.
type GTCLITimer struct {
	command string
	start   time.Time
}

// NewGTCLITimer creates a new timer for a gt CLI call.
func NewGTCLITimer(command string) *GTCLITimer {
	return &GTCLITimer{
		command: command,
		start:   time.Now(),
	}
}

// RecordSuccess records a successful gt CLI call.
func (t *GTCLITimer) RecordSuccess() {
	duration := time.Since(t.start).Seconds()
	GTCLIDuration.WithLabelValues(t.command).Observe(duration)
	GTCLICallsTotal.WithLabelValues(t.command, ResultSuccess).Inc()
}

// RecordError records a failed gt CLI call.
func (t *GTCLITimer) RecordError() {
	duration := time.Since(t.start).Seconds()
	GTCLIDuration.WithLabelValues(t.command).Observe(duration)
	GTCLICallsTotal.WithLabelValues(t.command, ResultError).Inc()
}

// UpdateRigPhase updates the rig phase gauge.
func UpdateRigPhase(phase string, count float64) {
	RigPhaseGauge.WithLabelValues(phase).Set(count)
}

// UpdatePolecatPhase updates the polecat phase gauge for a rig.
func UpdatePolecatPhase(rig, phase string, count float64) {
	PolecatPhaseGauge.WithLabelValues(rig, phase).Set(count)
}

// UpdateConvoyPhase updates the convoy phase gauge.
func UpdateConvoyPhase(phase string, count float64) {
	ConvoyPhaseGauge.WithLabelValues(phase).Set(count)
}

// RefineryMergeTimer tracks merge operation timing for a specific rig.
type RefineryMergeTimer struct {
	rig   string
	start time.Time
}

// NewRefineryMergeTimer creates a new timer for a refinery merge operation.
func NewRefineryMergeTimer(rig string) *RefineryMergeTimer {
	return &RefineryMergeTimer{
		rig:   rig,
		start: time.Now(),
	}
}

// RecordSuccess records a successful merge.
func (t *RefineryMergeTimer) RecordSuccess() {
	duration := time.Since(t.start).Seconds()
	RefineryMergeDuration.WithLabelValues(t.rig).Observe(duration)
	RefineryMergeTotal.WithLabelValues(t.rig, ResultSuccess).Inc()
}

// RecordError records a failed merge.
func (t *RefineryMergeTimer) RecordError() {
	duration := time.Since(t.start).Seconds()
	RefineryMergeDuration.WithLabelValues(t.rig).Observe(duration)
	RefineryMergeTotal.WithLabelValues(t.rig, ResultError).Inc()
}

// RecordConflict records a merge conflict for a rig.
func RecordConflict(rig string) {
	RefineryConflictsTotal.WithLabelValues(rig).Inc()
}

// UpdateQueueLength updates the merge queue length for a rig.
func UpdateQueueLength(rig string, length float64) {
	RefineryQueueLength.WithLabelValues(rig).Set(length)
}
