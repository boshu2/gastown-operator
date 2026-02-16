package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	// SecretStaleThresholdDays is the threshold for marking a secret as stale.
	// Secrets older than this trigger a Degraded condition.
	SecretStaleThresholdDays = 90
)

var (
	// gastownSecretAge tracks the age of secrets in days.
	// Labels: namespace, secret_name, secret_type
	gastownSecretAge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gastown_secret_age_seconds",
			Help: "Age of secrets in seconds since last sync timestamp annotation",
		},
		[]string{"namespace", "secret_name", "secret_type"},
	)
)

func init() {
	// Register metrics with the global prometheus registry
	metrics.Registry.MustRegister(gastownSecretAge)
}

// RecordSecretAge records the age of a secret in seconds.
// syncTimestamp is the RFC3339 timestamp from the secret's annotation.
func RecordSecretAge(namespace, secretName, secretType, syncTimestamp string) error {
	if syncTimestamp == "" {
		// No sync timestamp annotation - cannot compute age
		return nil
	}

	syncTime, err := time.Parse(time.RFC3339, syncTimestamp)
	if err != nil {
		return err
	}

	ageSeconds := time.Since(syncTime).Seconds()
	gastownSecretAge.WithLabelValues(namespace, secretName, secretType).Set(ageSeconds)
	return nil
}

// IsSecretStale checks if a secret is older than the stale threshold.
func IsSecretStale(syncTimestamp string) bool {
	if syncTimestamp == "" {
		return true // No timestamp = stale
	}

	syncTime, err := time.Parse(time.RFC3339, syncTimestamp)
	if err != nil {
		return true // Parse error = stale
	}

	ageDays := time.Since(syncTime).Hours() / 24
	return ageDays > SecretStaleThresholdDays
}
