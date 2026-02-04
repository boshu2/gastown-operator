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

package workqueue

import (
	"time"

	"golang.org/x/time/rate"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NewGastownRateLimiter creates a rate limiter with exponential backoff
// following the Cert-Manager pattern for production resilience.
//
// The rate limiter combines two strategies:
//   - Exponential backoff: 5ms base delay, doubling on each failure up to 5min max.
//     This prevents thundering herd problems when resources fail repeatedly.
//   - Bucket rate limiter: 10 items/sec steady state with burst capacity of 100.
//     This provides burst protection during high-load scenarios.
//
// The MaxOf combiner ensures the most conservative (longest) delay wins,
// providing both per-item backoff and global rate limiting.
func NewGastownRateLimiter() workqueue.TypedRateLimiter[reconcile.Request] {
	return workqueue.NewTypedMaxOfRateLimiter(
		// Exponential backoff: 5ms base, 5min max
		workqueue.NewTypedItemExponentialFailureRateLimiter[reconcile.Request](5*time.Millisecond, 5*time.Minute),
		// Bucket rate limiter: 10 items/sec, burst of 100
		&workqueue.TypedBucketRateLimiter[reconcile.Request]{Limiter: rate.NewLimiter(rate.Limit(10), 100)},
	)
}
